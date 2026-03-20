package compiler
import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)
// Statement translation methods never return AST nodes. Instead, they always
// append to the current block statement.
func (g *GoProgram) CompileStmt(node parser.Node) {
	switch n := node.(type) {
	case *parser.NoopNode:
		return
	case *parser.AssignmentNode:
		if len(n.Left) == 1 {
			if constant, ok := n.Left[0].(*parser.ConstantNode); ok {
				switch n.Right[0].Type() {
				case types.IntType, types.SymbolType, types.FloatType, types.StringType, types.BoolType:
					g.addConstant(g.it.Get(constant.Namespace+constant.Val), g.CompileExpr(n.Right[0]))
				default:
					g.addGlobalVar(g.it.Get(constant.Namespace+constant.Val), g.it.Get(n.Right[0].Type().GoType()), g.CompileExpr(n.Right[0]))
				}
				return
			}
		}
		g.CompileAssignmentNode(n)
	case *parser.ReturnNode:
		if n.Type() == nil {
			// Nil-typed return — compile for side effects only (e.g., void method calls)
			for _, v := range n.Val {
				g.CompileStmt(v)
			}
			return
		}
		if !n.Type().IsMultiple() {
			switch stmt := n.Val[0].(type) {
			case *parser.Condition, *parser.CaseNode:
				g.State.Push(InReturnStatement)
				defer g.State.Pop()
				g.CompileStmt(stmt)
			case *parser.MethodCall:
				if stmt.RequiresTransform() {
					g.State.Push(InReturnStatement)
					defer g.State.Pop()
					g.CompileStmt(stmt)
				}
			default:
				g.appendToCurrentBlock(&ast.ReturnStmt{
					Results: g.wrapOptionalReturn(g.mapToExprs(n.Val), n.Val),
				})
			}
		} else {
			g.appendToCurrentBlock(&ast.ReturnStmt{
				Results: g.wrapOptionalReturn(g.mapToExprs(n.Val), n.Val),
			})
		}
	case *parser.Condition:
		if n.TypeGuard {
			return // runtime type check — redundant in Go
		}
		cond := g.CompileExpr(n.Condition)
		// If the condition is an Optional or Proc type, compare against nil
		condType := n.Condition.Type()
		if ident, ok := n.Condition.(*parser.IdentNode); ok {
			if local := g.ScopeChain.ResolveVar(ident.Val); local != nil && local != parser.BadLocal && local.Type() != nil {
				condType = local.Type()
			}
		}
		nilCheck := false
		if _, isOpt := condType.(types.Optional); isOpt {
			nilCheck = true
		} else if _, isProc := condType.(*types.Proc); isProc {
			nilCheck = true
		} else if condType == types.FuncType {
			nilCheck = true
		}
		if nilCheck {
			cond = bst.Binary(cond, token.NEQ, g.it.Get("nil"))
		}
		// Remove conditional entirely if boolean value of cond expr is known at compile time
		if justBool, ok := cond.(*ast.Ident); ok && (justBool.Name == "true" || justBool.Name == "false") {
			if justBool.Name == "true" {
				g.appendToCurrentBlock(g.CompileBlockStmt(n.True).List...)
			} else {
				g.appendToCurrentBlock(g.CompileBlockStmt(n.False).List...)
			}
		} else {
			stmt := &ast.IfStmt{
				Cond: cond,
				Body: g.CompileBlockStmt(n.True),
			}
			if n.False != nil {
				stmt.Else = g.CompileBlockStmt(n.False)
			}
			g.appendToCurrentBlock(stmt)
		}
	case *parser.CaseNode:
		// Array-pattern case/when: desugar to if/elsif chain comparing elements pairwise.
		// Ruby: case [a, b]; when [true, true]; ... end
		// Go:   if a == true && b == true { ... } else if ...
		if arrVal, ok := n.Value.(*parser.ArrayNode); ok {
			g.compileArrayPatternCase(arrVal, n.Whens)
			break
		}
		stmt := &ast.SwitchStmt{}
		tag := g.CompileExpr(n.Value)
		if n.Value != nil && !n.RequiresExpansion {
			stmt.Tag = tag
		}
		g.newBlockStmt()
		for _, when := range n.Whens {
			list := []ast.Expr{}
			for _, arg := range when.Conditions {
				var expr ast.Expr
				if arg.Type().HasMethod("===") {
					transform := arg.Type().TransformAST(
						"===",
						g.CompileExpr(arg),
						[]types.TypeExpr{{n.Value.Type(), tag}},
						nil,
						g.it,
					)
					expr = transform.Expr
				} else {
					expr = g.CompileExpr(arg)
					if n.RequiresExpansion && isSimple(expr) {
						expr = bst.Binary(tag, token.EQL, expr)
					}
				}
				list = append(list, expr)
			}
			if len(list) == 0 {
				list = nil
			}
			if len(list) > 1 && n.RequiresExpansion {
				conds := list[0]
				for _, cond := range list[1:] {
					conds = bst.Binary(conds, token.LOR, cond)
				}
				list = []ast.Expr{conds}
			}
			g.appendToCurrentBlock(&ast.CaseClause{
				List: list,
				Body: g.CompileBlockStmt(when.Statements).List,
			})
		}
		stmt.Body = g.BlockStack.Peek()
		g.BlockStack.Pop()
		g.appendToCurrentBlock(stmt)
	case *parser.PatternMatchNode:
		g.compilePatternMatch(n)
	case *parser.MethodCall:
		if n.RequiresTransform() {
			stmtContext := g.State.Peek() != InReturnStatement
			var transform types.Transform
			if stmtContext {
				transform = g.TransformMethodCallStmt(n)
			} else {
				transform = g.TransformMethodCall(n)
			}
			// Rewrite hash-accessed mutations: h.Get(k) = append(...) → h.Set(k, append(...))
			if ba, ok := n.Receiver.(*parser.BracketAccessNode); ok {
				if _, isHash := ba.Composite.Type().(types.Hash); isHash {
					transform.Stmts = g.rewriteHashGetAssigns(transform.Stmts)
				}
			}
			g.appendToCurrentBlock(transform.Stmts...)
			if len(transform.Finalizers) > 0 {
				g.Finalizers = append(g.Finalizers, transform.Finalizers...)
			}
			if g.State.Peek() == InReturnStatement && n.Type() != types.NilType {
				results := g.wrapOptionalReturn([]ast.Expr{transform.Expr}, []parser.Node{n})
				g.appendToCurrentBlock(&ast.ReturnStmt{
					Results: results,
				})
				// A Transform may yield only statements, in which case Expr could be
				// nil.  It could also return an expression solely for the purposes of
				// chaining, which in this case we don't need the Expr because we already
				// are expecting a statement. Ignore both these cases.
			} else if len(transform.Stmts) == 0 {
				if _, ok := transform.Expr.(*ast.CallExpr); ok {
					g.appendToCurrentBlock(&ast.ExprStmt{
						X: transform.Expr,
					})
				}
			}
		} else {
			g.appendToCurrentBlock(&ast.ExprStmt{
				X: g.CompileExpr(n),
			})
		}
	case *parser.SuperNode:
		g.appendToCurrentBlock(&ast.ExprStmt{
			X: g.CompileSuperNode(n),
		})
	case *parser.WhileNode:
		// Ruby while loops don't create a new scope, so variables first-assigned
		// inside the body must be hoisted to the enclosing scope (like ForInNode).
		hoistWhileLoopVars(n.Body, g)
		forStmt := &ast.ForStmt{
			Body: g.CompileBlockStmt(n.Body),
		}
		// `loop do...end` desugars to WhileNode with true condition — emit bare `for {}`
		if boolNode, ok := n.Condition.(*parser.BooleanNode); !ok || boolNode.Val != "true" {
			forStmt.Cond = g.CompileExpr(n.Condition)
		}
		g.appendToCurrentBlock(forStmt)
	case *parser.BreakNode:
		g.appendToCurrentBlock(&ast.BranchStmt{
			Tok: token.BREAK,
		})
	case *parser.NextNode:
		if n.Val != nil {
			// next <value> — emit as return + continue so that
			// map/collect transforms can rewrite the return to append.
			g.appendToCurrentBlock(&ast.ReturnStmt{
				Results: []ast.Expr{g.CompileExpr(n.Val)},
			})
			g.appendToCurrentBlock(&ast.BranchStmt{
				Tok: token.CONTINUE,
			})
		} else {
			g.appendToCurrentBlock(&ast.BranchStmt{
				Tok: token.CONTINUE,
			})
		}
	case *parser.AliasNode:
		// Handled in CompileClass — no-op here
	case *parser.BeginNode:
		g.CompileBeginNode(n)
	case *parser.ForInNode:
		// Ruby for-loops don't create a new variable scope, so we have to declare
		// the variable upfront and use an assignment rather than declaration operator
		// in the range statement
		var outerScope []ast.Expr
		for _, v := range n.For {
			name := v.(*parser.IdentNode).Val
			outerScope = append(outerScope, g.it.Get(name))
			g.appendToCurrentBlock(&ast.DeclStmt{
				Decl: bst.Declare(token.VAR, g.it.Get(name), g.it.Get(v.Type().GoType())),
			})
		}
		rangeExpr := g.CompileExpr(n.In)
		if _, ok := n.In.Type().(types.Hash); ok {
			rangeExpr = bst.Call(rangeExpr, "All")
		}
		loop := &ast.RangeStmt{
			Tok:  token.ASSIGN,
			X:    rangeExpr,
			Body: g.CompileBlockStmt(n.Body),
		}
		if _, ok := n.In.Type().(types.Hash); ok {
			loop.Key, loop.Value = outerScope[0], outerScope[1]
		} else {
			loop.Key, loop.Value = g.it.Get("_"), outerScope[0]
		}
		g.appendToCurrentBlock(loop)
	case *parser.InfixExpressionNode:
		rcvrType := n.Left.Type()
		if spec, ok := rcvrType.GetMethodSpec(n.Operator); ok && spec.TransformStmtAST != nil {
			rcvr := g.CompileExpr(n.Left)
			argExprs := []types.TypeExpr{{Expr: g.CompileExpr(n.Right), Type: n.Right.Type()}}
			transform := spec.TransformStmtAST(types.TypeExpr{rcvrType, rcvr}, argExprs, nil, g.it)
			g.AddImports(transform.Imports...)
			g.appendToCurrentBlock(transform.Stmts...)
			return
		}
		rcvr := g.CompileExpr(n.Left)
		transform := g.getTransform(nil, rcvr, rcvrType, n.Operator, parser.ArgsNode{n.Right}, nil, false)
		// Rewrite hash-accessed mutations: h.Get(k) = append(...) → h.Set(k, ...)
		if ba, ok := n.Left.(*parser.BracketAccessNode); ok {
			if _, isHash := ba.Composite.Type().(types.Hash); isHash {
				transform.Stmts = g.rewriteHashGetAssigns(transform.Stmts)
			}
		}
		g.appendToCurrentBlock(transform.Stmts...)
		if len(transform.Stmts) == 0 {
			if _, ok := transform.Expr.(*ast.Ident); !ok {
				g.appendToCurrentBlock(&ast.ExprStmt{X: transform.Expr})
			}
		}
	default:
		expr := g.CompileExpr(n)
		// A single ident being returned here means we've prepended statements and
		// a transform has supplied an ident for potential chained operations. If
		// we got here, we're not going to make further calls on this object, so
		// skip it.
		if _, ok := expr.(*ast.Ident); !ok {
			g.appendToCurrentBlock(&ast.ExprStmt{
				X: expr,
			})
		}
	}
}
// selectorUsesIdent returns true if expr contains pkg.Method where pkg matches name.
// Handles both proper SelectorExpr nodes and dotted ident names (e.g. "pkg.Func").
func selectorUsesIdent(expr ast.Expr, name string) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return strings.HasPrefix(e.Name, name+".")
	case *ast.SelectorExpr:
		if ident, ok := e.X.(*ast.Ident); ok && ident.Name == name {
			return true
		}
		return selectorUsesIdent(e.X, name)
	case *ast.CallExpr:
		if selectorUsesIdent(e.Fun, name) {
			return true
		}
		for _, arg := range e.Args {
			if selectorUsesIdent(arg, name) {
				return true
			}
		}
	case *ast.BinaryExpr:
		return selectorUsesIdent(e.X, name) || selectorUsesIdent(e.Y, name)
	case *ast.UnaryExpr:
		return selectorUsesIdent(e.X, name)
	case *ast.ParenExpr:
		return selectorUsesIdent(e.X, name)
	}
	return false
}

func (g *GoProgram) CompileAssignmentNode(node *parser.AssignmentNode) {
	//TODO write test case specifically for multiple return values in branches of conditional statement
	// Handle conditional expressions (if/else, case/when) on the RHS.
	// We declare the variable outside the control flow statement and assign
	// inside each branch.
	if rhs, isCondExpr := node.Right[0].(*parser.Condition); isCondExpr {
		g.declareCondAssignmentVars(node)
		defer func() { g.State.Pop(); g.CurrentLhs = nil }()
		g.CompileStmt(rhs)
		return
	}
	if rhs, isCaseExpr := node.Right[0].(*parser.CaseNode); isCaseExpr {
		g.declareCondAssignmentVars(node)
		defer func() { g.State.Pop(); g.CurrentLhs = nil }()
		g.CompileStmt(rhs)
		return
	}
	// Bracket assignment on facade types with []= method → delegate to TransformAST
	if ba, ok := node.Left[0].(*parser.BracketAssignmentNode); ok {
		if rcvrType := ba.Composite.Type(); rcvrType != nil {
			if _, isHash := rcvrType.(types.Hash); !isHash {
				if spec, hasSpec := rcvrType.GetMethodSpec("[]="); hasSpec {
					rcvr := g.CompileExpr(ba.Composite)
					var typeExprs []types.TypeExpr
					for _, arg := range ba.Args {
						typeExprs = append(typeExprs, types.TypeExpr{Expr: g.CompileExpr(arg), Type: arg.Type()})
					}
					rhs := g.CompileExpr(node.Right[0])
					typeExprs = append(typeExprs, types.TypeExpr{Expr: rhs, Type: node.Right[0].Type()})
					transform := spec.TransformAST(types.TypeExpr{Type: rcvrType, Expr: rcvr}, typeExprs, nil, g.it)
					g.AddImports(transform.Imports...)
					for _, s := range transform.Stmts {
						g.appendToCurrentBlock(s)
					}
					if transform.Expr != nil && len(transform.Stmts) == 0 {
						g.appendToCurrentBlock(&ast.ExprStmt{X: transform.Expr})
					}
					return
				}
			}
		}
	}
	// Methods that mutate slice params return them alongside the normal
	// return value. Expand `x = foo(arr, ...)` to `x, arr = foo(arr, ...)`.
	if len(node.Right) == 1 {
		if call, ok := node.Right[0].(*parser.MethodCall); ok && call.Method != nil && len(call.Method.MutatedSliceParams) > 0 {
			lhs := []ast.Expr{}
			for _, left := range node.Left {
				lhs = append(lhs, g.CompileExpr(left))
			}
			// Append the mutated slice args to the LHS
			for _, paramIdx := range call.Method.MutatedSliceParams {
				argNode := call.Args[paramIdx]
				lhs = append(lhs, g.CompileExpr(argNode))
			}
			rhs := g.CompileExpr(call)
			assignFn := bst.Define
			if node.Reassignment {
				assignFn = bst.Assign
			}
			g.appendToCurrentBlock(assignFn(lhs, []ast.Expr{rhs}))
			return
		}
	}

	// Hash/DefaultHash bracket assignment → h.Set(key, val) or h[key] = val for native maps
	if ba, ok := node.Left[0].(*parser.BracketAssignmentNode); ok {
		if h, isHash := ba.Composite.Type().(types.Hash); isHash {
			rcvr := g.CompileExpr(ba.Composite)
			key := g.CompileExpr(ba.Args[0])
			// Order-safe hashes use native map assignment
			if g.receiverIsOrderSafe(ba.Composite) {
				rhs := g.CompileExpr(node.Right[0])
				g.appendToCurrentBlock(bst.Assign(
					&ast.IndexExpr{X: rcvr, Index: key},
					rhs,
				))
				return
			}
			if h.HasDefault && node.OpAssignment {
				infix := node.Right[0].(*parser.InfixExpressionNode)
				rhs := g.CompileExpr(infix.Right)
				getExpr := bst.Call(rcvr, "Get", key)
				combined := bst.Binary(getExpr, bst.TokenForOp(infix.Operator), rhs)
				g.appendToCurrentBlock(&ast.ExprStmt{X: bst.Call(rcvr, "Set", key, combined)})
			} else {
				rhs := g.CompileExpr(node.Right[0])
				g.appendToCurrentBlock(&ast.ExprStmt{X: bst.Call(rcvr, "Set", key, rhs)})
			}
			return
		}
	}
	// Array bracket assignment with auto-grow: arr[i] = val where arr is a slice
	// Ruby arrays auto-extend on index assignment; Go slices panic on out-of-bounds.
	// Emit: if i >= len(arr) { arr = append(arr, make([]T, i-len(arr)+1)...) }; arr[i] = val
	if ba, ok := node.Left[0].(*parser.BracketAssignmentNode); ok {
		if arrType, isArray := ba.Composite.Type().(types.Array); isArray {
			rcvr := g.CompileExpr(ba.Composite)
			idx := g.CompileExpr(ba.Args[0])
			// Dereference Optional indices: check both direct type and scope-resolved type
			isOpt := false
			if _, ok := ba.Args[0].Type().(types.Optional); ok {
				isOpt = true
			} else if ident, ok := ba.Args[0].(*parser.IdentNode); ok {
				if local := g.ScopeChain.ResolveVar(ident.Val); local != nil && local != parser.BadLocal {
					if _, ok := local.Type().(types.Optional); ok {
						isOpt = true
					}
				}
			}
			if isOpt {
				idx = &ast.StarExpr{X: idx}
			}
			idx = g.negativeIndex(rcvr, ba.Args[0], idx)
			rhs := g.CompileExpr(node.Right[0])
			// Determine element type; if the array element is still AnyType,
			// fall back to the RHS type for a correct make() call.
			elemType := arrType.Element
			if elemType == types.AnyType || elemType == nil {
				if rhsType := node.Right[0].Type(); rhsType != nil && rhsType != types.AnyType {
					elemType = rhsType
				}
			}
			elemGoType := elemType.GoType()
			// if idx >= len(arr) { arr = append(arr, make([]T, idx-len(arr)+1)...) }
			growCall := &ast.CallExpr{
				Fun: g.it.Get("append"),
				Args: []ast.Expr{
					rcvr,
					&ast.CallExpr{
						Fun: g.it.Get("make"),
						Args: []ast.Expr{
							&ast.ArrayType{Elt: g.it.Get(elemGoType)},
							bst.Binary(bst.Binary(idx, token.SUB, bst.Call(nil, "len", rcvr)), token.ADD, bst.Int(1)),
						},
					},
				},
				Ellipsis: 1,
			}
			growStmt := &ast.IfStmt{
				Cond: bst.Binary(idx, token.GEQ, bst.Call(nil, "len", rcvr)),
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(rcvr, growCall),
					},
				},
			}
			g.appendToCurrentBlock(growStmt)
			// When array has Optional elements (e.g., sparse array returned
			// from function where consumer checks .nil?), wrap RHS with &.
			// Capture the value in a temp to avoid aliasing a loop variable
			// (all entries would share the same pointer as it mutates).
			if _, isOpt := elemType.(types.Optional); isOpt {
				tmp := g.it.New("v")
				g.appendToCurrentBlock(bst.Define(tmp, rhs))
				rhs = &ast.UnaryExpr{Op: token.AND, X: tmp}
			}
			g.appendToCurrentBlock(bst.Assign(
				&ast.IndexExpr{X: rcvr, Index: idx},
				rhs,
			))
			return
		}
	}
	var assignFunc bst.AssignFunc
	if node.OpAssignment {
		// operator-assignment can only have singular left and right hand sides ever
		infix := node.Right[0].(*parser.InfixExpressionNode)
		if intNode, ok := infix.Right.(*parser.IntNode); ok && intNode.Val == "1" && (infix.Operator == "+" || infix.Operator == "-") {
			op := token.INC
			if infix.Operator == "-" {
				op = token.DEC
			}
			g.appendToCurrentBlock(&ast.IncDecStmt{
				X:   g.CompileExpr(node.Left[0]),
				Tok: op,
			})
			return
		}
		// For ||= with Optional types, compile as: if x == nil { _v := rhs; x = &_v }
		// Also handles nil-default params where the ident was refined to the
		// concrete type during analysis but the param signature is *T.
		if infix.Operator == "||" {
			isOpt := false
			if _, ok := infix.Left.Type().(types.Optional); ok {
				isOpt = true
			} else if ident, ok := infix.Left.(*parser.IdentNode); ok && g.currentMethod != nil {
				for _, p := range g.currentMethod.Params {
					if p.Name == ident.Val && p.HasNilDefault() {
						isOpt = true
						break
					}
				}
			}
			if isOpt {
				g.suppressDeref = true
				lhs := g.CompileExpr(node.Left[0])
				g.suppressDeref = false
				rhs := g.CompileExpr(infix.Right)
				tmp := g.it.New("v")
				g.appendToCurrentBlock(&ast.IfStmt{
					Cond: bst.Binary(lhs, token.EQL, g.it.Get("nil")),
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							bst.Define(tmp, rhs),
							bst.Assign(lhs, &ast.UnaryExpr{Op: token.AND, X: tmp}),
						},
					},
				})
				return
			}
		}
		assignFunc = bst.OpAssign(infix.Operator)
		node = node.Copy().(*parser.AssignmentNode)
		node.Right = []parser.Node{infix.Right}
	} else if node.Reassignment {
		assignFunc = bst.Assign
	} else {
		assignFunc = bst.Define
	}
	// rhs must go first here for reason of generation of local variable names
	// in transforms
	// Set CurrentLhs for the duration of RHS compilation to enable type refinement
	prevLhs := g.CurrentLhs
	g.CurrentLhs = node.Left
	defer func() {
		g.CurrentLhs = prevLhs
	}()
	var rhs []ast.Expr
	for i, right := range node.Right {
		if _, ok := right.Type().(types.Array); ok && i == len(node.Right)-1 && len(node.Left)-1 > i {
			arr := g.CompileExpr(right)
			for j, left := range node.Left[i:len(node.Left)] {
				if _, ok := left.(*parser.SplatNode); ok {
					rhs = append(rhs, &ast.SliceExpr{
						X:    arr,
						Low:  bst.Int(j),
						High: bst.Call(nil, "len", arr),
					})
				} else {
					rhs = append(rhs, &ast.IndexExpr{
						X:     arr,
						Index: bst.Int(j),
					})
				}
			}
		} else {
			rhs = append(rhs, g.CompileExpr(right))
		}
	}
	// Detect variable shadowing: if LHS name matches a package selector in RHS,
	// remap the variable to avoid illegal Go like `lcs := lcs.Foo()`
	if !node.Reassignment && !node.OpAssignment {
		for i, rhsExpr := range rhs {
			if i < len(node.Left) {
				if ident, ok := node.Left[i].(*parser.IdentNode); ok {
					goName := bst.SanitizeName(ident.Val)
					if selectorUsesIdent(rhsExpr, goName) {
						// Ensure the name is registered before remapping,
						// so Remap actually creates a versioned name.
						g.it.Get(ident.Val)
						g.it.Remap(ident.Val)
					}
				}
			}
		}
	}
	var lhs []ast.Expr
	for i, left := range node.Left {
		if call, ok := left.(*parser.MethodCall); ok {
			if call.Setter {
				// attr_accessor setter: compile as field assignment
				field := bst.Dot(g.CompileExpr(call.Receiver), strings.Title(strings.TrimSuffix(call.MethodName, "=")))
				g.appendToCurrentBlock(bst.Assign(field, rhs[i]))
			} else {
				g.CompileStmt(call)
			}
			rhs = append(rhs[:i], rhs[i+1:]...)
		} else {
			lhs = append(lhs, g.CompileExpr(left))
		}
	}
	tautological := true
	if len(lhs) != len(rhs) {
		tautological = false
	}
	for i, left := range lhs {
		if rh, ok := rhs[i].(*ast.Ident); ok {
			if lh, ok := left.(*ast.Ident); ok {
				if rh.Name != lh.Name {
					tautological = false
					break
				}
			} else {
				tautological = false
				break
			}
		} else {
			tautological = false
			break
		}
	}
	if !tautological {
		// When declaring a variable with nil RHS and the variable has been
		// refined to a concrete type, emit `var x T` instead of `x := nil`
		// so Go knows the type (untyped nil is invalid in short declarations).
		if !node.Reassignment && !node.OpAssignment && len(node.Right) == 1 {
			if _, isNil := node.Right[0].(*parser.NilNode); isNil {
				emitted := false
				for i, left := range node.Left {
					if ident, ok := left.(*parser.IdentNode); ok {
						var resolvedType types.Type
						if local := g.ScopeChain.ResolveVar(ident.Val); local != nil && local.Type() != nil && local.Type() != types.AnyType && local.Type() != types.NilType {
							resolvedType = local.Type()
						}
						// Also check the ident node's own type — the analysis may
						// have refined it in a block scope that's no longer on the chain.
						if resolvedType == nil && ident.Type() != nil && ident.Type() != types.AnyType && ident.Type() != types.NilType {
							resolvedType = ident.Type()
						}
						if resolvedType != nil {
							g.appendToCurrentBlock(&ast.DeclStmt{
								Decl: &ast.GenDecl{
									Tok: token.VAR,
									Specs: []ast.Spec{&ast.ValueSpec{
										Names: []*ast.Ident{lhs[i].(*ast.Ident)},
										Type:  g.it.Get(resolvedType.GoType()),
									}},
								},
							})
							// Register in compiler scope so later references
							// to this variable can resolve its Optional type.
							g.ScopeChain.Set(ident.Val, &parser.RubyLocal{})
							if local, ok := g.ScopeChain.Get(ident.Val); ok {
								local.(*parser.RubyLocal).SetType(resolvedType)
							}
							emitted = true
						}
					}
				}
				if emitted {
					return
				}
			}
		}
		// When reassigning a concrete value to an Optional variable, wrap in stdlib.Ptr[T]()
		// Skip wrapping if:
		//   - RHS is already Optional (e.g., method returning *int)
		//   - RHS type is already a pointer type (e.g., SynthStruct with GoType "*Foo")
		//   - LHS was dereferenced (nil-default param refined to concrete type)
		if node.Reassignment {
			for i, left := range node.Left {
				if ident, ok := left.(*parser.IdentNode); ok && i < len(rhs) {
					if local := g.ScopeChain.ResolveVar(ident.Val); local != nil {
						if opt, ok := local.Type().(types.Optional); ok {
							if _, isNil := node.Right[i].(*parser.NilNode); !isNil {
								rhsType := node.Right[i].Type()
								// Skip if RHS is already Optional
								if _, rhsOpt := rhsType.(types.Optional); rhsOpt {
									continue
								}
								// Skip if RHS type already compiles to a pointer
								// (e.g., SynthStruct whose GoType is "*Foo")
								if rhsType != nil && strings.HasPrefix(rhsType.GoType(), "*") {
									continue
								}
								// Skip if LHS was dereferenced — the compiled LHS
								// is *ident, so RHS must be the inner type, not a pointer.
								if i < len(lhs) {
									if _, isStar := lhs[i].(*ast.StarExpr); isStar {
										continue
									}
								}
								rhs[i] = g.wrapPtr(rhs[i], opt.Element)
							}
						}
					}
				}
			}
		}
		g.appendToCurrentBlock(assignFunc(lhs, rhs))
	}
}

func (g *GoProgram) declareCondAssignmentVars(node *parser.AssignmentNode) {
	specs := []ast.Spec{}
	for _, left := range node.Left {
		identNode, isIdent := left.(*parser.IdentNode)
		if isIdent {
			localName := identNode.Val
			t := g.ScopeChain.ResolveVar(localName)
			if t.Type() == nil {
				panic(fmt.Sprintf("Attempted to resolve '%s' in '%s' but got no type on %#v", localName, node, t))
			}
			name := g.it.New(localName)
			specs = append(specs, &ast.ValueSpec{
				Names: []*ast.Ident{name},
				Type:  g.it.Get(t.Type().GoType()),
			})
		} else {
			panic("attempted to assign to LHS type other than ident")
		}
	}
	decl := &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok:   token.VAR,
			Specs: specs,
		},
	}
	g.appendToCurrentBlock(decl)
	g.State.Push(InCondAssignment)
	g.CurrentLhs = node.Left
}

func (g *GoProgram) CompileBeginNode(node *parser.BeginNode) {
	g.newBlockStmt()

	// Emit ensure defer first (runs LAST due to LIFO = after rescue, matching Ruby)
	if node.EnsureBody != nil {
		ensureBlock := g.CompileBlockStmt(node.EnsureBody)
		g.appendToCurrentBlock(&ast.DeferStmt{
			Call: &ast.CallExpr{
				Fun: &ast.FuncLit{
					Type: &ast.FuncType{Params: &ast.FieldList{}},
					Body: ensureBlock,
				},
			},
		})
	}

	// Emit rescue defer second (runs FIRST due to LIFO = before ensure, matching Ruby)
	if len(node.RescueClauses) > 0 {
		hasTypedClauses := false
		for _, clause := range node.RescueClauses {
			if len(clause.ExceptionTypes) > 0 {
				hasTypedClauses = true
				break
			}
		}

		r := g.it.New("r")

		var rescueBodyStmts []ast.Stmt
		if hasTypedClauses {
			rescueBodyStmts = g.compileTypedRescue(node.RescueClauses, r)
		} else {
			rescueBodyStmts = g.compileSimpleRescue(node.RescueClauses, r)
		}
		recoverIf := &ast.IfStmt{
			Init: bst.Define(r, bst.Call(nil, "recover")),
			Cond: bst.Binary(r, token.NEQ, g.it.Get("nil")),
			Body: &ast.BlockStmt{List: rescueBodyStmts},
		}

		g.appendToCurrentBlock(&ast.DeferStmt{
			Call: &ast.CallExpr{
				Fun: &ast.FuncLit{
					Type: &ast.FuncType{Params: &ast.FieldList{}},
					Body: &ast.BlockStmt{List: []ast.Stmt{recoverIf}},
				},
			},
		})
	}

	// Compile begin body
	for _, stmt := range node.Body {
		g.CompileStmt(stmt)
	}

	iifeBody := g.BlockStack.Peek()
	g.BlockStack.Pop()

	// Wrap in IIFE
	g.appendToCurrentBlock(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Body: iifeBody,
			},
		},
	})
}

func (g *GoProgram) compileSimpleRescue(clauses []*parser.RescueClause, r *ast.Ident) []ast.Stmt {
	var stmts []ast.Stmt
	for _, clause := range clauses {
		bodyBlock := g.CompileBlockStmt(clause.Body)
		if clause.ExceptionVar != "" && rescueVarUsed(clause.Body, clause.ExceptionVar) {
			e := g.it.New(clause.ExceptionVar)
			stmts = append(stmts, bst.Define(e, &ast.TypeAssertExpr{
				X:    r,
				Type: ast.NewIdent("error"),
			}))
		}
		stmts = append(stmts, bodyBlock.List...)
	}
	return stmts
}

func (g *GoProgram) compileTypedRescue(clauses []*parser.RescueClause, r *ast.Ident) []ast.Stmt {
	g.AddImports("github.com/redneckbeard/thanos/stdlib")

	// Determine the switch variable name. Use the first rescue variable name
	// that's actually referenced; this pre-registers it in the ident tracker so
	// body compilation generates matching references.
	var switchVarName string
	for _, clause := range clauses {
		if clause.ExceptionVar != "" && rescueVarUsed(clause.Body, clause.ExceptionVar) {
			switchVarName = clause.ExceptionVar
			break
		}
	}
	var switchVar *ast.Ident
	if switchVarName != "" {
		switchVar = g.it.New(switchVarName)
	}

	// Build type switch: switch [e :=] r.(type) { case *stdlib.X: ... }
	g.newBlockStmt()

	hasCatchAll := false
	for _, clause := range clauses {
		var caseList []ast.Expr
		if len(clause.ExceptionTypes) > 0 {
			for _, exType := range clause.ExceptionTypes {
				caseList = append(caseList, &ast.StarExpr{
					X: bst.Dot("stdlib", exType),
				})
			}
		} else {
			hasCatchAll = true
		}

		bodyBlock := g.CompileBlockStmt(clause.Body)
		g.appendToCurrentBlock(&ast.CaseClause{
			List: caseList,
			Body: bodyBlock.List,
		})
	}

	// If no catch-all rescue, re-panic on unmatched errors
	if !hasCatchAll {
		g.appendToCurrentBlock(&ast.CaseClause{
			Body: []ast.Stmt{
				&ast.ExprStmt{X: bst.Call(nil, "panic", r)},
			},
		})
	}

	switchBody := g.BlockStack.Peek()
	g.BlockStack.Pop()

	typeSwitch := &ast.TypeSwitchStmt{
		Body: switchBody,
	}
	if switchVar != nil {
		typeSwitch.Assign = bst.Define(switchVar, &ast.TypeAssertExpr{X: r})
	} else {
		typeSwitch.Assign = &ast.ExprStmt{X: &ast.TypeAssertExpr{X: r}}
	}

	return []ast.Stmt{typeSwitch}
}

func rescueVarUsed(stmts parser.Statements, name string) bool {
	for _, stmt := range stmts {
		if nodeReferencesVar(stmt, name) {
			return true
		}
	}
	return false
}

// rewriteHashGetAssigns rewrites assignment statements like
//   h.Get(key) = append(h.Get(key), val)
// to
//   h.Set(key, append(h.Get(key), val))
// This is needed because Go function returns aren't addressable on the LHS.
func (g *GoProgram) rewriteHashGetAssigns(stmts []ast.Stmt) []ast.Stmt {
	result := make([]ast.Stmt, 0, len(stmts))
	for _, stmt := range stmts {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 {
			result = append(result, stmt)
			continue
		}
		call, ok := assign.Lhs[0].(*ast.CallExpr)
		if !ok {
			result = append(result, stmt)
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Get" || len(call.Args) != 1 {
			result = append(result, stmt)
			continue
		}
		// Rewrite: rcvr.Get(key) = rhs → rcvr.Set(key, rhs)
		setCall := bst.Call(sel.X, "Set", call.Args[0], assign.Rhs[0])
		result = append(result, &ast.ExprStmt{X: setCall})
	}
	return result
}

func nodeReferencesVar(node parser.Node, name string) bool {
	switch n := node.(type) {
	case *parser.IdentNode:
		return n.Val == name
	case *parser.MethodCall:
		if n.Receiver != nil && nodeReferencesVar(n.Receiver, name) {
			return true
		}
		for _, arg := range n.Args {
			if nodeReferencesVar(arg, name) {
				return true
			}
		}
	}
	return false
}

// hoistWhileLoopVars scans a while loop body for variables first-assigned inside
// the loop. Since Ruby while loops don't create a new scope, these variables must
// be declared in the enclosing Go scope to be accessible after the loop.
func hoistWhileLoopVars(stmts parser.Statements, g *GoProgram) {
	hoisted := map[string]bool{}
	hoistWalk(stmts, hoisted, g)
}

func hoistWalk(stmts parser.Statements, hoisted map[string]bool, g *GoProgram) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *parser.AssignmentNode:
			if !s.Reassignment && !s.OpAssignment && !s.SetterCall {
				for i, left := range s.Left {
					if ident, ok := left.(*parser.IdentNode); ok {
						// Use RHS type since LHS ident may not be typed yet for first-assigns
						var rhsType types.Type
						if i < len(s.Right) {
							rhsType = s.Right[i].Type()
						}
						if !hoisted[ident.Val] && rhsType != nil && !containsAnyType(rhsType) {
							hoisted[ident.Val] = true
							g.appendToCurrentBlock(&ast.DeclStmt{
								Decl: bst.Declare(token.VAR, g.it.Get(ident.Val), g.it.Get(rhsType.GoType())),
							})
							s.Reassignment = true
						}
					}
				}
			}
		case *parser.WhileNode:
			hoistWalk(s.Body, hoisted, g)
		case *parser.Condition:
			hoistWalk(s.True, hoisted, g)
			if s.False != nil {
				if cond, ok := s.False.(*parser.Condition); ok {
					hoistWalk(cond.True, hoisted, g)
				}
			}
		}
	}
}

// compileArrayPatternCase desugars `case [a, b]; when [x, y]; ...` into an
// if/elsif chain: `if a == x && b == y { ... } else if ...`. This is needed
// because Go cannot switch on slices.
func (g *GoProgram) compileArrayPatternCase(arrVal *parser.ArrayNode, whens []*parser.WhenNode) {
	// Compile case value elements once into temp vars to avoid re-evaluation
	valExprs := make([]ast.Expr, len(arrVal.Args))
	for i, elem := range arrVal.Args {
		valExprs[i] = g.CompileExpr(elem)
	}

	var ifStmt *ast.IfStmt
	var lastIf *ast.IfStmt

	for _, when := range whens {
		body := g.CompileBlockStmt(when.Statements)

		if len(when.Conditions) == 0 {
			// else clause
			if lastIf != nil {
				lastIf.Else = body
			} else {
				// case with only else — just emit the body
				g.appendToCurrentBlock(body.List...)
			}
			continue
		}

		// Build condition: valExprs[0] == cond[0] && valExprs[1] == cond[1] && ...
		var cond ast.Expr
		for _, condArg := range when.Conditions {
			var elemCond ast.Expr
			if condArr, ok := condArg.(*parser.ArrayNode); ok {
				// Array when: compare element by element
				for i, condElem := range condArr.Args {
					if i >= len(valExprs) {
						break
					}
					eq := bst.Binary(valExprs[i], token.EQL, g.CompileExpr(condElem))
					if elemCond == nil {
						elemCond = eq
					} else {
						elemCond = bst.Binary(elemCond, token.LAND, eq)
					}
				}
			} else {
				// Non-array when condition — compare whole value (fallback)
				elemCond = g.CompileExpr(condArg)
			}
			if elemCond != nil {
				if cond == nil {
					cond = elemCond
				} else {
					cond = bst.Binary(cond, token.LOR, elemCond)
				}
			}
		}

		clause := &ast.IfStmt{
			Cond: cond,
			Body: body,
		}
		if ifStmt == nil {
			ifStmt = clause
			lastIf = clause
		} else {
			lastIf.Else = clause
			lastIf = clause
		}
	}

	if ifStmt != nil {
		g.appendToCurrentBlock(ifStmt)
	}
}

func containsAnyType(t types.Type) bool {
	if t == types.AnyType {
		return true
	}
	if arr, ok := t.(types.Array); ok {
		return arr.Element == types.AnyType
	}
	if h, ok := t.(types.Hash); ok {
		return h.Key == types.AnyType || h.Value == types.AnyType
	}
	return false
}

func (g *GoProgram) methodReturnsOptional() bool {
	if g.currentMethod == nil {
		return false
	}
	_, isOpt := g.currentMethod.ReturnType().(types.Optional)
	return isOpt
}

// wrapOptionalReturn wraps return expressions in stdlib.Ptr[T]() when the
// current method returns Optional(T) and the expression is a concrete value.
func (g *GoProgram) wrapOptionalReturn(exprs []ast.Expr, nodes []parser.Node) []ast.Expr {
	if g.currentMethod == nil {
		return exprs
	}
	retType := g.currentMethod.ReturnType()
	opt, isOpt := retType.(types.Optional)
	if !isOpt {
		return exprs
	}
	for i, expr := range exprs {
		// Don't wrap nil returns
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
			continue
		}
		if i < len(nodes) {
			nodeType := nodes[i].Type()
			// Don't wrap if the node is already Optional-typed
			if _, nodeOpt := nodeType.(types.Optional); nodeOpt {
				continue
			}
			// Don't wrap if the node's type already compiles to a pointer
			// (e.g., SynthStruct with GoType "*Foo")
			if nodeType != nil && strings.HasPrefix(nodeType.GoType(), "*") {
				continue
			}
		}
		exprs[i] = g.wrapPtr(expr, opt.Element)
	}
	return exprs
}

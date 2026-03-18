package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"math"
	"strconv"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

// interfaceType is a helper type that generates interface{} for empty arrays
type interfaceType struct{}

func (t *interfaceType) GoType() string                                                             { return "interface{}" }
func (t *interfaceType) String() string                                                             { return "interface{}" }
func (t *interfaceType) Equals(other types.Type) bool                                               { return false }
func (t *interfaceType) IsComposite() bool                                                          { return false }
func (t *interfaceType) IsMultiple() bool                                                           { return false }
func (t *interfaceType) ClassName() string                                                          { return "" }
func (t *interfaceType) HasMethod(m string) bool                                                    { return false }
func (t *interfaceType) MethodReturnType(m string, b types.Type, args []types.Type) (types.Type, error) { return nil, nil }
func (t *interfaceType) GetMethodSpec(m string) (types.MethodSpec, bool)                           { return types.MethodSpec{}, false }
func (t *interfaceType) BlockArgTypes(m string, args []types.Type) []types.Type                    { return nil }
func (t *interfaceType) TransformAST(m string, rcvr ast.Expr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
	return types.Transform{}
}

// inferEmptyArrayType attempts to infer the element type of an empty array from context
func (g *GoProgram) inferEmptyArrayType(arrayNode *parser.ArrayNode) types.Type {
	// This is a simplified heuristic - in practice, you'd want more sophisticated analysis
	// For now, default to string type for empty arrays used with join()
	return types.StringType
}

// Expression translation methods _do_ return AST Nodes because of the
// specificity of where they have to be inserted. Any additional statements can
// be prepended before returning.
func (g *GoProgram) CompileExpr(node parser.Node) ast.Expr {
	switch n := node.(type) {
	case *parser.InfixExpressionNode:
		return g.TransformInfixExpressionNode(n)
	case *parser.MethodCall:
		// Safe navigation operator: x&.method compiles to nil-guarded call
		if n.Op == "&." {
			if opt, ok := n.Receiver.Type().(types.Optional); ok {
				innerType := opt.Element
				rcvrExpr := g.CompileExpr(n.Receiver)

				// Declare result var as the Optional return type
				result := g.it.New("result")
				resultType := n.Type().(types.Optional)
				resultDecl := &ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{result},
								Type:  g.it.Get(resultType.GoType()),
							},
						},
					},
				}

				// Dereference receiver: *x
				derefRcvr := &ast.StarExpr{X: rcvrExpr}

				// Get the transform for the inner type's method
				var argExprs []types.TypeExpr
				for _, a := range n.Args {
					argExprs = append(argExprs, types.TypeExpr{Expr: g.CompileExpr(a), Type: a.Type()})
				}
				transform := innerType.TransformAST(n.MethodName, derefRcvr, argExprs, nil, g.it)
				g.AddImports(transform.Imports...)
				g.localizeExpr(transform.Expr)

				// Build the if-body: run transform stmts, then result = &val
				ifBody := append([]ast.Stmt{}, transform.Stmts...)
				tmp := g.it.New("v")
				ifBody = append(ifBody,
					bst.Define(tmp, transform.Expr),
					bst.Assign(result, &ast.UnaryExpr{Op: token.AND, X: tmp}),
				)

				g.appendToCurrentBlock(resultDecl, &ast.IfStmt{
					Cond: bst.Binary(rcvrExpr, token.NEQ, g.it.Get("nil")),
					Body: &ast.BlockStmt{List: ifBody},
				})
				return result
			}
		}
		if n.RequiresTransform() {
			// Special case: Hash.new — need to use the variable's refined type
			if n.MethodName == "new" {
				if _, isClass := n.Receiver.Type().(*types.Class); isClass {
					if h, isHash := n.Type().(types.Hash); isHash && h.HasDefault {
						return g.CompileHashNew(n, h)
					}
				}
			}
			transform := g.TransformMethodCall(n)
			g.appendToCurrentBlock(transform.Stmts...)
			return transform.Expr
		} else if n.Getter {
			return bst.Dot(g.CompileExpr(n.Receiver), strings.Title(n.MethodName))
		} else if n.Setter {
			return bst.Dot(g.CompileExpr(n.Receiver), strings.Title(strings.TrimSuffix(n.MethodName, "=")))
		}
		// defined?(expr) compiles to true — if we got here, it exists
		if n.MethodName == "defined?" {
			return g.it.Get("true")
		}
		// block_given? compiles to blk != nil
		if n.MethodName == "block_given?" && n.Receiver == nil {
			return bst.Binary(g.it.Get("blk"), token.NEQ, g.it.Get("nil"))
		}
		// yield desugars to blk.call(args...) — compile as blk(args...)
		if n.MethodName == "call" {
			if ident, ok := n.Receiver.(*parser.IdentNode); ok {
				args := []ast.Expr{}
				for _, a := range n.Args {
					args = append(args, g.CompileExpr(a))
				}
				return bst.Call(nil, ident.Val, args...)
			}
		}
		if n.Method == nil {
			panic("Method not set on MethodCall " + n.String())
		}
		args := types.UnwrapTypeExprs(g.CompileArgs(n, n.Args))
		if n.Block != nil {
			funcType := &ast.FuncType{
				Params: &ast.FieldList{
					List: g.GetFuncParams(n.Block.Params),
				},
				Results: &ast.FieldList{
					List: g.GetReturnType(n.Block.Body.ReturnType),
				},
			}
			args = append(args, &ast.FuncLit{
				Type: funcType,
				Body: g.CompileBlockStmt(n.Block.Body.Statements),
			})
		} else if n.Method != nil && n.Method.Block != nil {
			// Method expects a block but call site doesn't provide one — pass nil
			args = append(args, g.it.Get("nil"))
		}
		call := bst.Call(nil, strings.Title(n.MethodName), args...)
		if n.HasSplat() {
			call.Ellipsis = 1
		}
		return call
	case *parser.IdentNode:
		if n.MethodCall != nil {
			return g.CompileExpr(n.MethodCall)
		}
		ident := g.it.Get(n.Val)
		// Dereference nil-default params that were refined to concrete types.
		// The Go param is *T but after ||= the analysis refined the local to T.
		if !g.suppressDeref && g.currentMethod != nil {
			for _, p := range g.currentMethod.Params {
				if p.Name == n.Val && p.HasNilDefault() {
					if _, isOpt := n.Type().(types.Optional); !isOpt {
						return &ast.StarExpr{X: ident}
					}
				}
			}
		}
		return ident
	case *parser.IVarNode:
		ivar := n.NormalizedVal()
		if n.IVar().Readable && n.IVar().Writeable {
			ivar = strings.Title(ivar)
		}
		return &ast.SelectorExpr{
			X:   g.currentRcvr,
			Sel: g.it.Get(ivar),
		}
	case *parser.CVarNode:
		return g.it.Get(g.cvarGoName(n))
	case *parser.GVarNode:
		return g.it.Get(n.NormalizedVal())
	case *parser.NilNode:
		return g.it.Get("nil")
	case *parser.BooleanNode:
		return g.it.Get(n.Val)
	case *parser.IntNode:
		return bst.Int(n.Val)
	case *parser.Float64Node:
		return &ast.BasicLit{
			Kind:  token.FLOAT,
			Value: n.Val,
		}
	case *parser.ImaginaryNode:
		// Ruby 3i → Go 3i (complex128 literal)
		return &ast.BasicLit{
			Kind:  token.IMAG,
			Value: n.Val,
		}
	case *parser.RationalNode:
		// Ruby 3r → Go stdlib.NewRationalFromInt(3)
		g.AddImports("github.com/redneckbeard/thanos/stdlib")
		val := strings.TrimSuffix(n.Val, "r")
		return bst.Call("stdlib", "NewRationalFromInt", bst.Int(val))
	case *parser.SymbolNode:
		return bst.String(n.Val[1:])
	case *parser.StringNode:
		return g.CompileStringNode(n)

	case *parser.ArrayNode:
		// Tuple arrays (heterogeneous literals) cannot be compiled as Go slices.
		// They are only valid in contexts that consume them element-by-element
		// (e.g., string % formatting). If we reach here, the context doesn't support it.
		if _, isTuple := n.Type().(*types.Tuple); isTuple {
			panic(fmt.Sprintf("line %d: Heterogeneous array (Tuple) used in a context that requires a homogeneous collection", n.LineNo()))
		}
		// Handle empty arrays with AnyType
		arrayType := n.Type().(types.Array)
		elementType := arrayType.Element

		// Check if element type is Optional (array contains nil)
		_, isOptional := elementType.(types.Optional)

		elements := []ast.Expr{}
		var splat ast.Expr
		if isOptional {
			// Predeclare temp vars for non-nil values so we can take their address
			var tmpNames []*ast.Ident
			var tmpValues []ast.Expr
			for _, arg := range n.Args {
				if _, isNil := arg.(*parser.NilNode); isNil {
					elements = append(elements, g.it.Get("nil"))
				} else if s, ok := arg.(*parser.SplatNode); ok {
					splat = g.CompileExpr(s)
				} else {
					tmp := g.it.New("v")
					tmpNames = append(tmpNames, tmp)
					tmpValues = append(tmpValues, g.CompileExpr(arg))
					elements = append(elements, &ast.UnaryExpr{Op: token.AND, X: tmp})
				}
			}
			if len(tmpNames) > 0 {
				lhs := make([]ast.Expr, len(tmpNames))
				for i, name := range tmpNames {
					lhs[i] = name
				}
				g.appendToCurrentBlock(bst.Define(lhs, tmpValues))
			}
		} else {
			for _, arg := range n.Args {
				if s, ok := arg.(*parser.SplatNode); ok {
					splat = g.CompileExpr(s)
				} else {
					elements = append(elements, g.CompileExpr(arg))
				}
			}
		}

		// If still AnyType, check if the scope has a refined type for the LHS variable
		if elementType == types.AnyType && len(elements) == 0 {
			refined := false
			if g.CurrentLhs != nil {
				if ident, ok := g.CurrentLhs[0].(*parser.IdentNode); ok {
					if local := g.ScopeChain.ResolveVar(ident.Val); local != nil && local.Type() != nil {
						if arr, ok := local.Type().(types.Array); ok && arr.Element != types.AnyType {
							elementType = arr.Element
							refined = true
						}
					}
				}
			}
			if !refined {
				elementType = &interfaceType{}
			}
		}

		arr := &ast.CompositeLit{
			Type: &ast.ArrayType{
				Elt: g.it.Get(elementType.GoType()),
			},
			Elts: elements,
		}
		if splat != nil {
			call := bst.Call(nil, "append", arr, splat)
			call.Ellipsis = 1
			return call
		} else {
			return arr
		}
	case *parser.HashNode:
		hashType := n.Type().(types.Hash)
		// Check if the target variable is order-safe (can use native map)
		if g.hashLhsIsOrderSafe() {
			keyType := hashType.Key.GoType()
			valType := hashType.Value.GoType()
			elts := []ast.Expr{}
			for _, pair := range n.Pairs {
				var key ast.Expr
				if pair.Label != "" {
					key = bst.String(pair.Label)
				} else {
					key = g.CompileExpr(pair.Key)
				}
				elts = append(elts, &ast.KeyValueExpr{
					Key:   key,
					Value: g.CompileExpr(pair.Value),
				})
			}
			return &ast.CompositeLit{
				Type: g.it.Get(fmt.Sprintf("map[%s]%s", keyType, valType)),
				Elts: elts,
			}
		}
		g.AddImports("github.com/redneckbeard/thanos/stdlib")
		om := g.it.New("om")
		g.appendToCurrentBlock(bst.Define(om, bst.Call("stdlib", fmt.Sprintf("NewOrderedMap[%s, %s]", hashType.Key.GoType(), hashType.Value.GoType()))))
		for _, pair := range n.Pairs {
			var key ast.Expr
			if pair.Label != "" {
				key = bst.String(pair.Label)
			} else {
				key = g.CompileExpr(pair.Key)
			}
			g.appendToCurrentBlock(&ast.ExprStmt{X: bst.Call(om, "Set", key, g.CompileExpr(pair.Value))})
		}
		return om
	case *parser.BracketAccessNode:
		rcvr := g.CompileExpr(n.Composite)
		if n.Composite.Type() != nil && n.Composite.Type().HasMethod("[]") {
			// For order-safe hashes, use native map indexing
			if _, isHash := n.Composite.Type().(types.Hash); isHash && g.receiverIsOrderSafe(n.Composite) {
				return &ast.IndexExpr{X: rcvr, Index: g.CompileExpr(n.Args[0])}
			}
			transform := g.getTransform(nil, rcvr, n.Composite.Type(), "[]", n.Args, nil, false)
			g.appendToCurrentBlock(transform.Stmts...)
			return transform.Expr
		}
		if n.Composite.Type() == types.StringType {
			rcvr = bst.Call(nil, "[]rune", rcvr)
		}
		if r, ok := n.Args[0].(*parser.RangeNode); ok {
			rangeIndex := g.CompileRangeIndexNode(rcvr, r)
			if n.Composite.Type() == types.StringType {
				rangeIndex = bst.Call(nil, "string", rangeIndex)
			}
			return rangeIndex
		} else {
			idx := g.CompileExpr(n.Args[0])
			idx = g.negativeIndex(rcvr, n.Args[0], idx)
			var index ast.Expr = &ast.IndexExpr{
				X:     rcvr,
				Index: idx,
			}
			if n.Composite.Type() == types.StringType {
				index = bst.Call(nil, "string", index)
			}
			return index
		}
	case *parser.BracketAssignmentNode:
		rcvr := g.CompileExpr(n.Composite)
		idx := g.CompileExpr(n.Args[0])
		idx = g.negativeIndex(rcvr, n.Args[0], idx)
		return &ast.IndexExpr{
			X:     rcvr,
			Index: idx,
		}
	case *parser.SelfNode:
		return g.currentRcvr
	case *parser.ConstantNode:
		if predefined, ok := types.PredefinedConstants[n.Val]; ok {
			g.AddImports(predefined.Imports...)
			return predefined.Expr
		}
		return g.it.Get(g.localName(n.Namespace + n.Val))
	case *parser.ScopeAccessNode:
		return g.it.Get(n.ReceiverName() + n.Constant)
	case *parser.NotExpressionNode:
		if arg, ok := n.Arg.(*parser.InfixExpressionNode); ok && arg.Operator == "==" {
			eq := g.CompileExpr(arg).(*ast.BinaryExpr)
			eq.Op = token.NEQ
			return eq
		}
		return &ast.UnaryExpr{
			Op: token.NOT,
			X:  g.CompileExpr(n.Arg),
		}
	case *parser.RangeNode:
		g.AddImports("github.com/redneckbeard/thanos/stdlib")
		bounds := g.mapToExprs([]parser.Node{n.Lower, n.Upper})
		args := append(bounds, g.it.Get(strconv.FormatBool(n.Inclusive)))
		return &ast.CompositeLit{
			Type: &ast.IndexExpr{
				X: &ast.SelectorExpr{
					X:   g.it.Get("&stdlib"),
					Sel: g.it.Get("Range"),
				},
				Index: g.it.Get(n.Type().(types.Range).Element.GoType()),
			},
			Elts: args,
		}
	case *parser.SuperNode:
		return g.CompileSuperNode(n)
	case *parser.SplatNode:
		return g.CompileExpr(n.Arg)
	case *parser.LambdaNode:
		funcType := &ast.FuncType{
			Params: &ast.FieldList{
				List: g.GetFuncParams(n.Block.Params),
			},
		}
		if n.Block.Body.ReturnType != nil && n.Block.Body.ReturnType != types.NilType {
			funcType.Results = &ast.FieldList{
				List: g.GetReturnType(n.Block.Body.ReturnType),
			}
		}
		return &ast.FuncLit{
			Type: funcType,
			Body: g.CompileBlockStmt(n.Block.Body.Statements),
		}
	case *parser.Condition:
		// The following duplicates much of what is done for an assignment with a
		// conditional on the RHS, but we can't easily reuse that year because the local
		// var that gets generated won't be identifiable from this spot in the tree.
		name := g.it.New("cond")
		g.appendToCurrentBlock(&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{&ast.ValueSpec{
					Names: []*ast.Ident{name},
					Type:  g.it.Get(n.Type().GoType()),
				}},
			},
		})
		g.State.Push(InCondAssignment)
		g.CurrentLhs = []parser.Node{&parser.IdentNode{Val: name.Name}}
		g.CompileStmt(n)
		g.CurrentLhs = nil
		g.State.Pop()
		return name
	case *parser.AssignmentNode:
		// Chained assignment (e.g., a = b = 0): compile inner assignment as a
		// statement and return the RHS value as the expression.
		g.CompileAssignmentNode(n)
		// The last LHS variable is the expression value
		return g.CompileExpr(n.Left[len(n.Left)-1])
	default:
		return &ast.BadExpr{}
	}
}

func (g *GoProgram) CompileRangeIndexNode(rcvr ast.Expr, r *parser.RangeNode) ast.Expr {
	bounds := map[int]ast.Expr{}

	for i, bound := range []parser.Node{r.Lower, r.Upper} {
		if bound != nil {
			switch b := bound.(type) {
			case *parser.IntNode:
				// if it's a literal, we can just set up the slice
				x, _ := strconv.Atoi(b.Val)
				if x < 0 {
					boundExpr := &ast.BinaryExpr{
						X:  bst.Call(nil, "len", rcvr),
						Op: token.SUB,
					}
					if r.Inclusive && i == 1 {
						x += 1
					}
					boundExpr.Y = bst.Int(int(math.Abs(float64(x))))
					bounds[i] = boundExpr
				} else {
					if r.Inclusive && i == 1 {
						b.Val = strconv.Itoa(x + 1)
					}
					bounds[i] = g.CompileExpr(b)
				}
			case *parser.IdentNode:
				/*
					This case is much worse than a literal. What we need to build is
					something like this:

					   var lower, upper int
					   if foo < 0 {
					     lower = len(x) + foo
					   } else {
					     lower = foo
					   }

					We could avoid doing this for cases when a variable for the slice
					value is defined and initialized with a literal inside the current
					block, but that would make this code even more complicated.
				*/
				var local *ast.Ident
				if i == 0 {
					local = g.it.New("lower")
				} else {
					local = g.it.New("upper")
				}
				g.appendToCurrentBlock(&ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{&ast.ValueSpec{
							Names: []*ast.Ident{local},
							Type:  g.it.Get("int"),
						}},
					},
				})
				var rhs ast.Expr
				if r.Inclusive && i == 1 {
					rhs = bst.Binary(g.CompileExpr(b), token.ADD, bst.Int(1))
				} else {
					rhs = g.CompileExpr(b)
				}
				cond := &ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X:  g.CompileExpr(b),
						Y:  bst.Int(0),
						Op: token.LSS,
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							bst.Assign(local, &ast.BinaryExpr{
								X:  bst.Call(nil, "len", rcvr),
								Op: token.ADD,
								Y:  rhs,
							}),
						},
					},
					Else: bst.Assign(local, rhs),
				}
				g.appendToCurrentBlock(cond)
				bounds[i] = local
			}
		}
	}

	sliceExpr := &ast.SliceExpr{X: rcvr}
	for k, v := range bounds {
		if k == 0 {
			sliceExpr.Low = v
		} else {
			sliceExpr.High = v
		}
	}

	return sliceExpr
}

func (g *GoProgram) TransformInfixExpressionNode(node *parser.InfixExpressionNode) ast.Expr {
	// Pattern: h[key] || default → val, ok := h[key]; if !ok { val = default }
	if node.Operator == "||" {
		if ba, ok := node.Left.(*parser.BracketAccessNode); ok {
			if h, isHash := ba.Composite.Type().(types.Hash); isHash && !h.HasDefault {
				g.Warn(node.LineNo(), "h[key] || default compiles to ok-check pattern. Consider using h.fetch(key, default) for cleaner output.")
				rcvr := g.CompileExpr(ba.Composite)
				key := g.CompileExpr(ba.Args[0])
				def := g.CompileExpr(node.Right)
				val := g.it.New("val")
				ok := g.it.New("ok")
				var mapExpr ast.Expr
				if g.receiverIsOrderSafe(ba.Composite) {
					mapExpr = rcvr
				} else {
					mapExpr = bst.Dot(rcvr, "Data")
				}
				g.appendToCurrentBlock(
					bst.Define(
						[]ast.Expr{val, ok},
						[]ast.Expr{&ast.IndexExpr{X: mapExpr, Index: key}},
					),
					&ast.IfStmt{
						Cond: &ast.UnaryExpr{Op: token.NOT, X: ok},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								bst.Assign(val, def),
							},
						},
					},
				)
				return val
			}
		}
	}
	// String#% with a Tuple (heterogeneous array literal): splat elements into fmt.Sprintf
	if node.Operator == "%" && node.Left.Type() == types.StringType {
		if arr, ok := node.Right.(*parser.ArrayNode); ok {
			if _, isTuple := arr.Type().(*types.Tuple); isTuple {
				g.AddImports("fmt")
				fmtArgs := []ast.Expr{g.CompileExpr(node.Left)}
				for _, a := range arr.Args {
					fmtArgs = append(fmtArgs, g.CompileExpr(a))
				}
				return bst.Call("fmt", "Sprintf", fmtArgs...)
			}
		}
	}
	transform := g.getTransform(nil, g.CompileExpr(node.Left), node.Left.Type(), node.Operator, parser.ArgsNode{node.Right}, nil, false)
	// Rewrite hash-accessed mutations: h.Get(k) = append(...) → h.Set(k, ...)
	if ba, ok := node.Left.(*parser.BracketAccessNode); ok {
		if _, isHash := ba.Composite.Type().(types.Hash); isHash {
			transform.Stmts = g.rewriteHashGetAssigns(transform.Stmts)
		}
	}
	g.appendToCurrentBlock(transform.Stmts...)
	return transform.Expr
}

func (g *GoProgram) CompileStringNode(node *parser.StringNode) ast.Expr {
	// Non-interpolated strings: use GoString() directly
	if len(node.Interps) == 0 {
		str := &ast.BasicLit{
			Kind:  token.STRING,
			Value: node.GoString(),
		}
		if node.Kind == parser.SingleQuote || node.Kind == parser.DoubleQuote {
			return str
		}
		switch node.Kind {
		case parser.Regexp:
			g.AddImports("regexp")
			patt := globalIdents.New("patt")
			g.addGlobalVar(patt, nil, bst.Call("regexp", "MustCompile", str))
			return patt
		case parser.RawWords:
			g.AddImports("strings")
			return bst.Call("strings", "Fields", str)
		case parser.Words:
			return &ast.CompositeLit{
				Type: &ast.ArrayType{Elt: g.it.Get("string")},
				Elts: g.stringElements(node),
			}
		case parser.Exec, parser.RawExec:
			g.AddImports("os/exec")
			outputVariable := g.it.New("output")
			g.appendToCurrentBlock(bst.Define(
				[]ast.Expr{outputVariable, g.it.Get("_")},
				bst.Call(bst.Call("exec", "Command", g.stringElements(node)...), "Output"),
			))
			return bst.Call(nil, "string", outputVariable)
		default:
			return str
		}
	}

	// Interpolated strings: defer verb resolution to a post-pass so that
	// Remap type changes (and block param types set by transforms) are
	// visible when we pick format verbs.
	delim := `"`
	switch node.Kind {
	case parser.Regexp, parser.SingleQuote, parser.RawWords, parser.RawExec:
		delim = "`"
	}

	d := deferredSprintf{delim: delim}
	var args []ast.Expr

	for i, seg := range node.BodySegments {
		if interps, exists := node.Interps[i]; exists {
			for _, interp := range interps {
				compiled, fallback := g.compileInterpExpr(interp)
				d.pieces = append(d.pieces, fmtPiece{arg: compiled, fallback: fallback})
				args = append(args, compiled)
			}
		}
		escaped, _ := node.TranslateEscapes(seg)
		d.pieces = append(d.pieces, fmtPiece{text: escaped})
	}
	if trailingInterps, exists := node.Interps[len(node.BodySegments)]; exists {
		for _, interp := range trailingInterps {
			compiled, fallback := g.compileInterpExpr(interp)
			d.pieces = append(d.pieces, fmtPiece{arg: compiled, fallback: fallback})
			args = append(args, compiled)
		}
	}

	// Build initial format string with fallback verbs. The post-pass may
	// update verbs if the IdentTracker has better type info (e.g., after Remap).
	// We build it now so that other transforms (like puts adding \n) can
	// modify the BasicLit without the post-pass clobbering their changes.
	initialFmt := ""
	for _, p := range d.pieces {
		if p.arg != nil {
			initialFmt += p.fallback
		} else {
			initialFmt += p.text
		}
	}
	str := &ast.BasicLit{Kind: token.STRING, Value: delim + initialFmt + delim}
	d.fmtLit = str
	// Store the parent tracker (not the current block-scoped one) because
	// transforms that set types on block params operate on the parent.
	if len(g.TrackerStack) >= 2 {
		d.tracker = g.TrackerStack[len(g.TrackerStack)-2]
	} else {
		d.tracker = g.it
	}
	g.deferredInterps = append(g.deferredInterps, d)

	allArgs := append([]ast.Expr{str}, args...)
	g.AddImports("fmt")
	formatted := bst.Call("fmt", "Sprintf", allArgs...)

	switch node.Kind {
	case parser.Regexp:
		g.AddImports("regexp")
		patt := g.it.New("patt")
		g.appendToCurrentBlock(bst.Define(
			[]ast.Expr{patt, g.it.Get("_")},
			bst.Call("regexp", "Compile", formatted),
		))
		return patt
	case parser.Words:
		return &ast.CompositeLit{
			Type: &ast.ArrayType{Elt: g.it.Get("string")},
			Elts: g.stringElements(node),
		}
	case parser.Exec, parser.RawExec:
		g.AddImports("os/exec")
		outputVariable := g.it.New("output")
		g.appendToCurrentBlock(bst.Define(
			[]ast.Expr{outputVariable, g.it.Get("_")},
			bst.Call(bst.Call("exec", "Command", g.stringElements(node)...), "Output"),
		))
		return bst.Call(nil, "string", outputVariable)
	default:
		return formatted
	}
}

// compileInterpExpr compiles an interpolation expression and returns the
// compiled expr plus a fallback verb derived from the Ruby AST type. Float
// expressions are wrapped with stdlib.FormatFloat and get fallback "%s".
func (g *GoProgram) compileInterpExpr(interp parser.Node) (ast.Expr, string) {
	compiled := g.CompileExpr(interp)

	t := interp.Type()
	if t == types.FloatType {
		g.AddImports("github.com/redneckbeard/thanos/stdlib")
		return bst.Call("stdlib", "FormatFloat", compiled), "%s"
	}
	verb := types.FprintVerb(t)
	if verb == "" {
		panic(fmt.Sprintf("Unhandled type inference failure for interpolated value in string"))
	}
	return compiled, verb
}

// finalizeDeferredInterps runs after all compilation (including transforms
// and Remaps) to resolve format verbs using the IdentTracker's type info.
func (g *GoProgram) finalizeDeferredInterps() {
	for _, d := range g.deferredInterps {
		// Rebuild the core format string with resolved verbs.
		resolved := ""
		for _, p := range d.pieces {
			if p.arg != nil {
				resolved += resolveVerb(d.tracker, p.arg, p.fallback)
			} else {
				resolved += p.text
			}
		}

		// Preserve any suffix that other transforms appended to the BasicLit
		// (e.g., puts adds \n before the closing delimiter).
		current := d.fmtLit.Value
		// Build what the initial value was (before other transforms modified it)
		initial := d.delim
		for _, p := range d.pieces {
			if p.arg != nil {
				initial += p.fallback
			} else {
				initial += p.text
			}
		}
		initial += d.delim
		// The suffix is whatever was inserted between the initial content and closing delim
		suffix := ""
		if len(current) > len(initial) {
			suffix = current[len(initial)-len(d.delim) : len(current)-len(d.delim)]
		}

		d.fmtLit.Value = d.delim + resolved + suffix + d.delim
	}
}

// resolveVerb determines the format verb for an interpolation arg expression.
// It checks the stored IdentTracker for type info set by Remap or SetType,
// falling back to the Ruby-derived fallback verb.
func resolveVerb(tracker bst.IdentTracker, expr ast.Expr, fallback string) string {
	if ident, ok := expr.(*ast.Ident); ok {
		if goType := tracker.GoType(ident.Name); goType != "" {
			return verbFromGoType(goType)
		}
	}
	return fallback
}

func verbFromGoType(goType string) string {
	switch {
	case goType == "string" || strings.HasPrefix(goType, "[]string"):
		return "%s"
	case goType == "int" || strings.HasPrefix(goType, "[]int"):
		return "%d"
	case goType == "float64" || strings.HasPrefix(goType, "[]float64"):
		return "%f"
	case goType == "bool":
		return "%t"
	default:
		return "%v"
	}
}

func (g *GoProgram) stringElements(node *parser.StringNode) []ast.Expr {
	// Ruby interpolated words apply the splitting on whitespace _before_
	// interpolation. There's no sensible way to achieve this in Go, so we
	// leave the nonsense in the compiler and have output be a string slice
	// literal.
	var elements []ast.Expr

	for i, seg := range node.BodySegments {
		if interps, exists := node.Interps[i]; exists {
			for _, interp := range interps {
				compiled, fallback := g.compileInterpExpr(interp)
				elements = append(elements, bst.Call("fmt", "Sprintf", bst.String(fallback), compiled))
			}
		}
		for _, s := range strings.Fields(seg) {
			elements = append(elements, bst.String(s))
		}
	}
	if trailingInterps, exists := node.Interps[len(node.BodySegments)]; exists {
		for _, trailingInterp := range trailingInterps {
			compiled, fallback := g.compileInterpExpr(trailingInterp)
			elements = append(elements, bst.Call("fmt", "Sprintf", bst.String(fallback), compiled))
		}
	}

	return elements
}

func (g *GoProgram) CompileSuperNode(node *parser.SuperNode) ast.Expr {
	_, method, found := node.Class.GetAncestorMethod(node.Method.Name)
	if !found && node.Class.DataDefine {
		// Data.define super in initialize just sets fields — Go struct handles this.
		return g.it.Get("nil")
	}
	params := []*ast.Field{
		{
			Names: []*ast.Ident{g.currentRcvr},
			Type: &ast.StarExpr{
				X: g.it.Get(node.Class.Name()),
			},
		},
	}
	params = append(params, g.GetFuncParams(method.Params)...)
	superType := &ast.FuncType{
		Params: &ast.FieldList{
			List: params,
		},
		Results: &ast.FieldList{
			List: g.GetReturnType(method.ReturnType()),
		},
	}
	superVar := g.it.New("super")
	g.appendToCurrentBlock(bst.Define(superVar, &ast.FuncLit{
		Type: superType,
		Body: g.CompileBlockStmt(node.Inline()),
	}))
	args := []ast.Expr{g.currentRcvr}
	if len(node.Args) > 0 {
		args = append(args, g.mapToExprs(node.Args)...)
	} else {
		for _, p := range params[1:] {
			args = append(args, p.Names[0])
		}
	}
	return bst.Call(nil, superVar, args...)
}

func (g *GoProgram) CompileArgs(call *parser.MethodCall, args parser.ArgsNode) []types.TypeExpr {
	var (
		argExprs, splatArgs []types.TypeExpr
		seenKeywords        []string
	)

	doubleSplatArg := call.ExtractDoubleSplatArg()

	for i := 0; i < len(call.Method.Params); i++ {
		p, _ := call.Method.GetParam(i)
		switch p.Kind {
		case parser.Positional:
			argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileArg(args[i])})
		case parser.Named:
			if i >= len(args) {
				argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileArg(p.Default)})
			} else if _, ok := args[i].(*parser.KeyValuePair); ok {
				argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileArg(p.Default)})
			} else {
				compiled := g.CompileArg(args[i])
				if _, isOpt := p.Type().(types.Optional); isOpt {
					if _, isNil := args[i].(*parser.NilNode); !isNil {
						compiled = g.wrapPtr(compiled, p.Type().(types.Optional).Element)
					}
				}
				argExprs = append(argExprs, types.TypeExpr{p.Type(), compiled})
			}
		case parser.Keyword:

			/*
				The interplay of keyword arguments and keyword parameters is very messy
				when a method has a double splat parameter defined.

				* The argument could match a parameter by name, or it could be
				  relegated to the double splat;
				* The argument could match a parameter by name but be overridden in a
				  double-splatted argument;
				* The keyword parameter could have a default, and only be overridden in
				  a double-splatted argument, etc.

				For this reason, much of this logic is unfortunately split across two
				branches in this switch: here and the parser.DoubleSplat case below.
			*/

			if arg, err := args.FindByName(p.Name); err != nil {
				if doubleSplatArg != nil {
					argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileKeyFromDoubleSplatArg(p.Name, doubleSplatArg)})
					seenKeywords = append(seenKeywords, p.Name)
				} else {
					argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileArg(p.Default)})
				}
			} else {
				argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileArg(arg.(*parser.KeyValuePair).Value)})
				seenKeywords = append(seenKeywords, p.Name)
			}
		case parser.Splat:
			for _, arg := range call.SplatArgs() {
				if splat, ok := arg.(*parser.SplatNode); ok {
					/*
						Ruby allows you to destructure with a splat argument to a splat parameter, e.g.
						`foo(x, *y)` considers `x, *y` to be the single splat argument to `def foo(*arg); end`
						Go, on the other hand, does not allow this, so in the case above we have to:

						1. create an array of all non-splat args that correspond to a splat parameter
						2. append the splat arg to that array
						3. include the ellipsis after that

						This results in something ugly in the target like `append([]T{x}, y...)...`, but it works.
					*/
					if len(splatArgs) == 0 {
						arg = splat.Arg
					} else {
						var elements []ast.Expr
						for _, a := range splatArgs {
							elements = append(elements, a.Expr)
						}
						arr := &ast.CompositeLit{
							Type: &ast.ArrayType{
								Elt: g.it.Get(splatArgs[0].Type.GoType()),
							},
							Elts: elements,
						}
						appendCall := bst.Call(nil, "append", arr, g.CompileExpr(splat.Arg))
						appendCall.Ellipsis = 1
						splatArgs = []types.TypeExpr{
							{splat.Arg.Type(), appendCall},
						}
						break
					}
				}
				splatArgs = append(splatArgs, types.TypeExpr{arg.Type(), g.CompileArg(arg)})
			}
		case parser.DoubleSplat:
			keywordArgsWithoutParam := call.KeywordArgsForDoubleSplatParam()
			if doubleSplatArg != nil {
				if h, ok := doubleSplatArg.(*parser.HashNode); ok {
					keywordArgsWithoutParam.Merge(h)
					for _, k := range seenKeywords {
						keywordArgsWithoutParam.Delete(k)
					}
				} else {
					/*
						The very grossest of situations. If we've got a double splat
						parameter, and we've got a double splat argument that's not a hash,
						that means that somewhere in the target above this invocation we've
						got a local with a map assigned to it, and we need to use it as this
						argument. However, it may contain keys that belong to named
						parameters. So the target needs to initialize a new map, copy over
						all the keys that aren't named parameters, and use the new map as the
						argument.
					*/
					splatIdent := doubleSplatArg.(*parser.IdentNode)
					original := g.it.Get(splatIdent.Val)
					paramHash := g.it.New(original.Name + "_kwargs")
					g.appendToCurrentBlock(bst.Define(paramHash, g.CompileExpr(keywordArgsWithoutParam)))

					keywordParams := []ast.Expr{}

					for _, p := range call.Method.Params {
						if p.Kind == parser.Keyword {
							keywordParams = append(keywordParams, bst.String(p.Name))
						}
					}

					k, v := g.it.Get("k"), g.it.Get("v")

					var rangeExpr ast.Expr
					if g.receiverIsOrderSafe(splatIdent) {
						rangeExpr = original
					} else {
						rangeExpr = bst.Call(original, "All")
					}

					loop := &ast.RangeStmt{
						Key:   k,
						Value: v,
						Tok:   token.DEFINE,
						X:     rangeExpr,
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.SwitchStmt{
									Tag: k,
									Body: &ast.BlockStmt{
										List: []ast.Stmt{
											&ast.CaseClause{
												List: keywordParams,
											},
											&ast.CaseClause{
												Body: []ast.Stmt{
													&ast.ExprStmt{X: bst.Call(paramHash, "Set", k, v)},
												},
											},
										},
									},
								},
							},
						},
					}

					g.appendToCurrentBlock(loop)

					argExprs = append(argExprs, types.TypeExpr{keywordArgsWithoutParam.Type(), paramHash})
					break
				}
			}
			argExprs = append(argExprs, types.TypeExpr{keywordArgsWithoutParam.Type(), g.CompileExpr(keywordArgsWithoutParam)})
		}
	}
	return append(argExprs, splatArgs...)
}

func (g *GoProgram) CompileArg(node parser.Node) ast.Expr {
	if assignment, ok := node.(*parser.AssignmentNode); ok {
		g.CompileStmt(assignment)
		return g.CompileExpr(assignment.Left[0])
	}
	return g.CompileExpr(node)
}

// wrapPtr wraps an expression in stdlib.Ptr[T](...) to convert a value to a pointer.
func (g *GoProgram) wrapPtr(expr ast.Expr, elemType types.Type) ast.Expr {
	return &ast.CallExpr{
		Fun:  g.it.Get(fmt.Sprintf("stdlib.Ptr[%s]", elemType.GoType())),
		Args: []ast.Expr{expr},
	}
}

func (g *GoProgram) CompileKeyFromDoubleSplatArg(key string, node parser.Node) ast.Expr {
	switch n := node.(type) {
	case *parser.HashNode:
		for _, kv := range n.Pairs {
			if kv.Label == key {
				return g.CompileExpr(kv.Value)
			}
		}
	case *parser.IdentNode:
		if g.isOrderSafe(n.Val) {
			return &ast.IndexExpr{
				X:     g.it.Get(n.Val),
				Index: bst.String(key),
			}
		}
		return &ast.IndexExpr{
			X:     bst.Dot(g.it.Get(n.Val), "Data"),
			Index: bst.String(key),
		}
	}
	return &ast.BadExpr{}
}

// negativeIndex translates Ruby negative array/string indices to Go.
// For literal negative ints: arr[-1] → arr[len(arr)-1]
// For variables: arr[i] → arr[stdlib.NegIndex(i, len(arr))] (only when type could be negative)
func (g *GoProgram) negativeIndex(rcvr ast.Expr, node parser.Node, idx ast.Expr) ast.Expr {
	if intNode, ok := node.(*parser.IntNode); ok && strings.HasPrefix(intNode.Val, "-") {
		absVal := intNode.Val[1:]
		return bst.Binary(bst.Call(nil, "len", rcvr), token.SUB, bst.Int(absVal))
	}
	return idx
}

// CompileHashNew generates Go code for Hash.new(default) and Hash.new { |h, k| ... }
func (g *GoProgram) CompileHashNew(n *parser.MethodCall, h types.Hash) ast.Expr {
	// Get the refined type from the LHS variable if available
	if g.CurrentLhs != nil {
		if ident, ok := g.CurrentLhs[0].(*parser.IdentNode); ok {
			if local := g.ScopeChain.ResolveVar(ident.Val); local != nil {
				if refined, ok := local.Type().(types.Hash); ok && refined.HasDefault {
					h = refined
				}
			}
		}
	}
	// If still unrefined, try the enclosing method's return type. This handles
	// cases like Hash.new { ... }.tap { ... } where there's no LHS variable.
	if (h.Key == types.AnyType || h.Value == types.AnyType) && g.currentMethod != nil {
		if retHash, ok := g.currentMethod.ReturnType().(types.Hash); ok && retHash.HasDefault {
			h = retHash
		}
	}
	keyType := h.Key.GoType()
	valType := h.Value.GoType()
	g.AddImports("github.com/redneckbeard/thanos/stdlib")

	if len(n.Args) > 0 {
		// Hash.new(0) → stdlib.NewDefaultHashWithValue[K, V](0)
		arg := g.CompileExpr(n.Args[0])
		return &ast.CallExpr{
			Fun:  g.it.Get(fmt.Sprintf("stdlib.NewDefaultHashWithValue[%s, %s]", keyType, valType)),
			Args: []ast.Expr{arg},
		}
	}
	if n.Block != nil {
		// Hash.new { |h, k| h[k] = [] } → stdlib.NewDefaultHash[K, V](func(m *stdlib.OrderedMap[K, V], k K) V { ... })
		blk := g.BuildBlock(n.Block)
		// Fix empty composite literals in block body to use the refined value type.
		// The block is compiled before type refinement, so e.g. []interface{}{}
		// needs to become []string{} to match the refined hash value type.
		g.fixBlockLiteralTypes(blk.Statements, valType)
		omType := fmt.Sprintf("*stdlib.OrderedMap[%s, %s]", keyType, valType)
		m := blk.Args[0]
		k := blk.Args[1]
		// The block body may end with a return statement from block compilation;
		// remove it and add return m.Data[k]
		finalIdx := len(blk.Statements) - 1
		if _, ok := blk.Statements[finalIdx].(*ast.ReturnStmt); ok {
			blk.Statements = blk.Statements[:finalIdx]
		}
		blk.Statements = append(blk.Statements, &ast.ReturnStmt{
			Results: []ast.Expr{&ast.IndexExpr{X: bst.Dot(m, "Data"), Index: k}},
		})
		funcLit := &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{
					List: []*ast.Field{
						{Names: []*ast.Ident{m.(*ast.Ident)}, Type: g.it.Get(omType)},
						{Names: []*ast.Ident{k.(*ast.Ident)}, Type: g.it.Get(keyType)},
					},
				},
				Results: &ast.FieldList{
					List: []*ast.Field{
						{Type: g.it.Get(valType)},
					},
				},
			},
			Body: &ast.BlockStmt{List: blk.Statements},
		}
		return &ast.CallExpr{
			Fun:  g.it.Get(fmt.Sprintf("stdlib.NewDefaultHash[%s, %s]", keyType, valType)),
			Args: []ast.Expr{funcLit},
		}
	}
	// Plain Hash.new → stdlib.NewOrderedMap[K, V]()
	return bst.Call("stdlib", fmt.Sprintf("NewOrderedMap[%s, %s]", keyType, valType))
}

// fixBlockLiteralTypes walks block statements and replaces empty composite
// literals whose type contains "interface{}" with the correct refined type.
// This handles the case where a block like { |h,k| h[k] = [] } is compiled
// before the value type is refined from subsequent usage.
func (g *GoProgram) fixBlockLiteralTypes(stmts []ast.Stmt, valType string) {
	for _, stmt := range stmts {
		ast.Inspect(stmt, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for i, arg := range call.Args {
				if cl, ok := arg.(*ast.CompositeLit); ok && len(cl.Elts) == 0 {
					if at, ok := cl.Type.(*ast.ArrayType); ok {
						if ident, ok := at.Elt.(*ast.Ident); ok && ident.Name == "interface{}" {
							call.Args[i] = &ast.CompositeLit{
								Type: g.it.Get(valType),
							}
						}
					}
				}
			}
			return true
		})
	}
}

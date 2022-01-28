package compiler

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

// Statement translation methods never return AST nodes. Instead, they always
// append to the current block statement.
func (g *GoProgram) CompileStmt(node parser.Node) {
	switch n := node.(type) {
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
					Results: g.mapToExprs(n.Val),
				})
			}
		} else {
			g.appendToCurrentBlock(&ast.ReturnStmt{
				Results: g.mapToExprs(n.Val),
			})
		}
	case *parser.Condition:
		cond := g.CompileExpr(n.Condition)
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
	case *parser.MethodCall:
		if n.RequiresTransform() {
			transform := g.TransformMethodCall(n)
			g.appendToCurrentBlock(transform.Stmts...)
			if g.State.Peek() == InReturnStatement && n.Type() != types.NilType {
				g.appendToCurrentBlock(&ast.ReturnStmt{
					Results: []ast.Expr{transform.Expr},
				})
				// A Transform may yield only statements, in which case Expr could be
				// nil.  It could also return an expression solely for the purposes of
				// chaining, which in this case we don't need the Expr because we already
				// are expecting a statement. Ignore both these cases.
			} else if _, ok := transform.Expr.(*ast.CallExpr); ok {
				stmt := &ast.ExprStmt{
					X: transform.Expr,
				}
				g.appendToCurrentBlock(stmt)
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
		g.appendToCurrentBlock(&ast.ForStmt{
			Cond: g.CompileExpr(n.Condition),
			Body: g.CompileBlockStmt(n.Body),
		})
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

func (g *GoProgram) CompileAssignmentNode(node *parser.AssignmentNode) {
	//TODO write test case specifically for multiple return values in branches of conditional statement
	if cond, ok := node.Right[0].(*parser.Condition); ok {

		// Here we handle the impedance mismatch between Ruby conditional
		// expressions and Go conditional statments.

		// We declare the variable outside the if statement, getting its type from
		// the local scope
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
		// Now before translating the branches of the conditional, we transition
		// the compiler's state.  This way subsequent steps will know to use an
		// assignment rather than definition operator.
		g.State.Push(InCondAssignment)
		g.CurrentLhs = node.Left
		defer func() {
			g.State.Pop()
			g.CurrentLhs = nil
		}()
		g.CompileStmt(cond)
		return
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
	var rhs []ast.Expr
	if _, ok := node.Right[0].Type().(types.Array); ok && len(node.Right) == 1 && len(node.Left) > 1 {
		arr := g.CompileExpr(node.Right[0])
		for i := range node.Left {
			rhs = append(rhs, &ast.IndexExpr{
				X:     arr,
				Index: bst.Int(i),
			})
		}
	} else {
		rhs = g.mapToExprs(node.Right)
	}
	var lhs []ast.Expr
	for i, left := range node.Left {
		if call, ok := left.(*parser.MethodCall); ok {
			// If we have a setter call, we have to ignore the corresponding rhs
			// value and append this to the current block statement.

			// TODO this is not right for cases with multiple setters and locals in
			// the same assignment. Will need to work backwards through the lhs bits.
			// Should write a failing test first.
			g.CompileStmt(call)
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
		g.appendToCurrentBlock(assignFunc(lhs, rhs))
	}
}

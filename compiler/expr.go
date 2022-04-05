package compiler

import (
	"go/ast"
	"go/token"
	"math"
	"strconv"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

// Expression translation methods _do_ return AST Nodes because of the
// specificity of where they have to be inserted. Any additional statements can
// be prepended before returning.
func (g *GoProgram) CompileExpr(node parser.Node) ast.Expr {
	switch n := node.(type) {
	case *parser.InfixExpressionNode:
		return g.TransformInfixExpressionNode(n)
	case *parser.MethodCall:
		if n.RequiresTransform() {
			transform := g.TransformMethodCall(n)
			g.appendToCurrentBlock(transform.Stmts...)
			return transform.Expr
		} else if n.Getter {
			return bst.Dot(g.CompileExpr(n.Receiver), strings.Title(n.MethodName))
		} else if n.Setter {
			return bst.Dot(g.CompileExpr(n.Receiver), strings.Title(strings.TrimSuffix(n.MethodName, "=")))
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
		return g.it.Get(n.Val)
	case *parser.IVarNode:
		ivar := n.NormalizedVal()
		if n.IVar().Readable && n.IVar().Writeable {
			ivar = strings.Title(ivar)
		}
		return &ast.SelectorExpr{
			X:   g.currentRcvr,
			Sel: g.it.Get(ivar),
		}
	case *parser.BooleanNode:
		return g.it.Get(n.Val)
	case *parser.IntNode:
		return bst.Int(n.Val)
	case *parser.Float64Node:
		return &ast.BasicLit{
			Kind:  token.FLOAT,
			Value: n.Val,
		}
	case *parser.SymbolNode:
		return bst.String(n.Val[1:])
	case *parser.StringNode:
		return g.CompileStringNode(n)

	case *parser.ArrayNode:
		elements := []ast.Expr{}
		var splat ast.Expr
		for _, arg := range n.Args {
			if s, ok := arg.(*parser.SplatNode); ok {
				splat = g.CompileExpr(s)
			} else {
				elements = append(elements, g.CompileExpr(arg))
			}
		}
		arr := &ast.CompositeLit{
			Type: &ast.ArrayType{
				Elt: g.it.Get(n.Type().(types.Array).Element.GoType()),
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
		elements := []ast.Expr{}
		for _, pair := range n.Pairs {
			var key ast.Expr
			if pair.Label != "" {
				key = bst.String(pair.Label)
			} else {
				key = g.CompileExpr(pair.Key)
			}
			elements = append(elements, &ast.KeyValueExpr{
				Key:   key,
				Value: g.CompileExpr(pair.Value),
			})
		}
		return &ast.CompositeLit{
			Type: &ast.MapType{
				Key:   g.it.Get(hashType.Key.GoType()),
				Value: g.it.Get(hashType.Value.GoType()),
			},
			Elts: elements,
		}
	case *parser.BracketAccessNode:
		rcvr := g.CompileExpr(n.Composite)
		if n.Composite.Type().HasMethod("[]") {
			transform := g.getTransform(nil, rcvr, n.Composite.Type(), "[]", n.Args, nil)
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
			var index ast.Expr = &ast.IndexExpr{
				X:     rcvr,
				Index: g.CompileExpr(n.Args[0]),
			}
			if n.Composite.Type() == types.StringType {
				index = bst.Call(nil, "string", index)
			}
			return index
		}
	case *parser.BracketAssignmentNode:
		return &ast.IndexExpr{
			X:     g.CompileExpr(n.Composite),
			Index: g.CompileExpr(n.Args[0]),
		}
	case *parser.SelfNode:
		return g.currentRcvr
	case *parser.ConstantNode:
		if predefined, ok := types.PredefinedConstants[n.Val]; ok {
			g.AddImports(predefined.Imports...)
			return predefined.Expr
		}
		return g.it.Get(n.Namespace + n.Val)
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
	transform := g.getTransform(nil, g.CompileExpr(node.Left), node.Left.Type(), node.Operator, parser.ArgsNode{node.Right}, nil)
	g.appendToCurrentBlock(transform.Stmts...)
	return transform.Expr
}

func (g *GoProgram) CompileStringNode(node *parser.StringNode) ast.Expr {
	// We don't want to use bst.String here, because node.GoString() will already
	//correctly surround the string
	str := &ast.BasicLit{
		Kind:  token.STRING,
		Value: node.GoString(),
	}
	if len(node.Interps) == 0 && (node.Kind == parser.SingleQuote || node.Kind == parser.DoubleQuote) {
		return str
	}
	args := []ast.Expr{str}
	for _, a := range node.OrderedInterps() {
		args = append(args, g.CompileExpr(a))
	}

	g.AddImports("fmt")

	formatted := bst.Call("fmt", "Sprintf", args...)
	switch node.Kind {
	case parser.Regexp:
		g.AddImports("regexp")
		var patt *ast.Ident
		if len(node.Interps) == 0 {
			// Ideally, people aren't regenerating regexes based on user input, so we can compile them at init time
			patt = globalIdents.New("patt")
			g.addGlobalVar(patt, nil, bst.Call("regexp", "MustCompile", str))
		} else {
			// ...but if not, just do it inline and swallow the error for now
			patt = g.it.New("patt")
			g.appendToCurrentBlock(bst.Define(
				[]ast.Expr{patt, g.it.Get("_")},
				bst.Call("regexp", "Compile", formatted),
			))
		}
		return patt
	case parser.RawWords:
		g.AddImports("strings")
		return bst.Call("strings", "Fields", str)
	case parser.Words:
		return &ast.CompositeLit{
			Type: &ast.ArrayType{
				Elt: g.it.Get("string"),
			},
			Elts: g.stringElements(node),
		}
	case parser.Exec, parser.RawExec:
		g.AddImports("os/exec")
		outputVariable := g.it.New("output")
		g.appendToCurrentBlock(bst.Define(
			[]ast.Expr{outputVariable, g.it.Get("_")},
			bst.Call(
				bst.Call("exec", "Command", g.stringElements(node)...),
				"Output",
			),
		))
		return bst.Call(nil, "string", outputVariable)
	default:
		return formatted
	}
}

func (g *GoProgram) stringElements(node *parser.StringNode) []ast.Expr {
	// Ruby interpolated words apply the splitting on whitespace _before_
	// interpolation. There's no sensible way to achieve this in Go, so we
	// leave the nonsense in the compiler and have output be a string slice
	// literal.

	// The following is nearly exactly how we generate the format string, but
	// it is not immediately obvious how to DRY it out without obscuring the
	// intentions of the former.
	var elements []ast.Expr

	for i, seg := range node.BodySegments {
		if interps, exists := node.Interps[i]; exists {
			for _, interp := range interps {
				verb := types.FprintVerb(interp.Type())
				elements = append(elements, bst.Call("fmt", "Sprintf", bst.String(verb), g.CompileExpr(interp)))
			}
		}
		for _, s := range strings.Fields(seg) {
			elements = append(elements, bst.String(s))
		}
	}
	if trailingInterps, exists := node.Interps[len(node.BodySegments)]; exists {
		for _, trailingInterp := range trailingInterps {
			verb := types.FprintVerb(trailingInterp.Type())
			elements = append(elements, bst.Call("fmt", "Sprintf", bst.String(verb), g.CompileExpr(trailingInterp)))
		}
	}

	return elements
}

func (g *GoProgram) CompileSuperNode(node *parser.SuperNode) ast.Expr {
	_, method, _ := node.Class.GetAncestorMethod(node.Method.Name)
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
	var argExprs, splatArgs []types.TypeExpr

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
				argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileArg(args[i])})
			}
		case parser.Keyword:
			if arg, err := args.FindByName(p.Name); err != nil {
				argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileArg(p.Default)})
			} else {
				argExprs = append(argExprs, types.TypeExpr{p.Type(), g.CompileArg(arg.(*parser.KeyValuePair).Value)})
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

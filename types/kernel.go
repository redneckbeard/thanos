package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type kernel struct {
	*proto
}

var (
	KernelType  = kernel{newProto("Kernel", "", ClassRegistry)}
	KernelClass = NewClass("Kernel", "", KernelType, ClassRegistry)
)

func (t kernel) Equals(t2 Type) bool { return t == t2 }
func (t kernel) String() string      { return "kernel" }
func (t kernel) GoType() string      { return "n/a" }
func (t kernel) IsComposite() bool   { return false }

func (t kernel) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t kernel) BlockArgTypes(m string, args []Type) []Type {
	spec := t.proto.MustResolve(m, false)
	return spec.BlockArgs(t, args)
}

func (t kernel) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t kernel) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t kernel) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t kernel) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t kernel) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
	KernelType.Def("print", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{
					&ast.ExprStmt{
						X: bst.Call("fmt", "Print", UnwrapTypeExprs(args)...),
					},
				},
				Imports: []string{"fmt"},
			}
		},
	})
	KernelType.Def("puts", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stmts := []ast.Stmt{}
			imports := []string{"fmt"}
			// `puts` inserts newlines after every argument, so we have one print function call for each here
			for _, arg := range args {
				// For any args that are interpolated strings, at this point we've
				// already translated them to the appropriate C-style interpolated
				// string. It would be weird in Go to call out to fmt.Sprintf for an
				// arg to fmt.Println, so here we grab those nodes, change the method
				// call to Printf, and insert a newline into the end of the string.
				if call, ok := arg.Expr.(*ast.CallExpr); ok {
					if fname, hasReceiver := call.Fun.(*ast.SelectorExpr); hasReceiver && fname.Sel.Name == "Sprintf" {
						fname.Sel = it.Get("Printf")
						lit := call.Args[0].(*ast.BasicLit)
						lit.Value = lit.Value[:len(lit.Value)-1] + `\n"`
						stmts = append(stmts, &ast.ExprStmt{X: call})
						continue
					}
				}
				printArg := arg.Expr
				if _, isOpt := arg.Type.(Optional); isOpt {
					printArg = &ast.StarExpr{X: arg.Expr}
				}
				if arg.Type == FloatType {
					stmts = append(stmts, &ast.ExprStmt{
						X: bst.Call("fmt", "Println", bst.Call("stdlib", "FormatFloat", printArg)),
					})
					imports = append(imports, "github.com/redneckbeard/thanos/stdlib")
				} else {
					stmts = append(stmts, &ast.ExprStmt{
						X: bst.Call("fmt", "Println", printArg),
					})
				}
			}
			return Transform{
				Stmts:   stmts,
				Imports: imports,
			}
		},
	})
	KernelType.Def("gauntlet", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.BadExpr{},
			}
		},
	})
	KernelType.Def("raise", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var panicArg ast.Expr
			switch len(args) {
			case 1:
				// raise "message" → panic(&stdlib.RuntimeError{RubyError: stdlib.RubyError{Msg: msg}})
				panicArg = &ast.UnaryExpr{
					Op: token.AND,
					X: &ast.CompositeLit{
						Type: bst.Dot("stdlib", "RuntimeError"),
						Elts: []ast.Expr{
							&ast.KeyValueExpr{
								Key: it.Get("StandardError"),
								Value: &ast.CompositeLit{
									Type: bst.Dot("stdlib", "StandardError"),
									Elts: []ast.Expr{
										&ast.KeyValueExpr{
											Key: it.Get("RubyError"),
											Value: &ast.CompositeLit{
												Type: bst.Dot("stdlib", "RubyError"),
												Elts: []ast.Expr{
													&ast.KeyValueExpr{
														Key:   it.Get("Msg"),
														Value: args[0].Expr,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}
			case 2:
				// raise ErrorClass, "message" → panic(&stdlib.ErrorClass{...Msg: msg})
				className := "RuntimeError"
				if ident, ok := args[0].Expr.(*ast.Ident); ok {
					className = ident.Name
				}
				panicArg = buildExceptionLiteral(className, args[1].Expr, it)
			default:
				// bare raise → panic(&stdlib.RuntimeError{})
				panicArg = &ast.UnaryExpr{
					Op: token.AND,
					X: &ast.CompositeLit{
						Type: bst.Dot("stdlib", "RuntimeError"),
					},
				}
			}
			return Transform{
				Stmts: []ast.Stmt{
					&ast.ExprStmt{
						X: bst.Call(nil, "panic", panicArg),
					},
				},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	KernelType.Def("require", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{}
		},
	})
	KernelType.Def("require_relative", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{}
		},
	})
	KernelType.Def("warn", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{
					&ast.ExprStmt{
						X: bst.Call("fmt", "Fprintln", append([]ast.Expr{bst.Dot("os", "Stderr")}, UnwrapTypeExprs(args)...)...),
					},
				},
				Imports: []string{"fmt", "os"},
			}
		},
	})

	KernelType.Def("block_given?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// Compiled to blk != nil in compiler/expr.go
			return Transform{Expr: bst.Binary(it.Get("blk"), token.NEQ, it.Get("nil"))}
		},
	})
}

// buildExceptionLiteral creates &stdlib.ClassName{...{RubyError: stdlib.RubyError{Msg: msg}}}
func buildExceptionLiteral(className string, msgExpr ast.Expr, it bst.IdentTracker) ast.Expr {
	// Build the nested struct literal chain based on the exception hierarchy
	innermost := &ast.CompositeLit{
		Type: bst.Dot("stdlib", "RubyError"),
		Elts: []ast.Expr{
			&ast.KeyValueExpr{
				Key:   it.Get("Msg"),
				Value: msgExpr,
			},
		},
	}

	parent, hasParent := ExceptionParents[className]
	if !hasParent {
		// It's StandardError itself or unknown — wrap directly with RubyError
		return &ast.UnaryExpr{
			Op: token.AND,
			X: &ast.CompositeLit{
				Type: bst.Dot("stdlib", className),
				Elts: []ast.Expr{
					&ast.KeyValueExpr{
						Key:   it.Get("RubyError"),
						Value: innermost,
					},
				},
			},
		}
	}

	// For types like ArgumentError whose parent is StandardError:
	// &stdlib.ArgumentError{StandardError: stdlib.StandardError{RubyError: stdlib.RubyError{Msg: msg}}}
	parentLit := &ast.CompositeLit{
		Type: bst.Dot("stdlib", parent),
		Elts: []ast.Expr{
			&ast.KeyValueExpr{
				Key:   it.Get("RubyError"),
				Value: innermost,
			},
		},
	}

	return &ast.UnaryExpr{
		Op: token.AND,
		X: &ast.CompositeLit{
			Type: bst.Dot("stdlib", className),
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key:   it.Get(parent),
					Value: parentLit,
				},
			},
		},
	}
}

package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Float struct {
	*proto
}

var FloatType = Float{newProto("Float", "Numeric", ClassRegistry)}

var FloatClass = NewClass("Float", "Numeric", FloatType, ClassRegistry)

func (t Float) Equals(t2 Type) bool { return t == t2 }
func (t Float) String() string      { return "FloatType" }
func (t Float) GoType() string      { return "float64" }
func (t Float) IsComposite() bool   { return false }

func (t Float) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Float) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Float) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Float) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Float) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Float) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Float) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
	FloatType.Def("ceil", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun:  it.Get("int"),
					Args: []ast.Expr{bst.Call("math", "Ceil", rcvr.Expr)},
				},
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("floor", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun:  it.Get("int"),
					Args: []ast.Expr{bst.Call("math", "Floor", rcvr.Expr)},
				},
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("round", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun:  it.Get("int"),
					Args: []ast.Expr{bst.Call("math", "Round", rcvr.Expr)},
				},
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("to_i", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun:  it.Get("int"),
					Args: []ast.Expr{rcvr.Expr},
				},
			}
		},
	})
	FloatType.Def("to_s", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strconv", "FormatFloat", rcvr.Expr, &ast.BasicLit{Kind: token.CHAR, Value: "'f'"}, &ast.UnaryExpr{Op: token.SUB, X: bst.Int(1)}, bst.Int(64)),
				Imports: []string{"strconv"},
			}
		},
	})
	FloatType.Def("zero?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.EQL, &ast.BasicLit{Kind: token.FLOAT, Value: "0.0"}),
			}
		},
	})
	FloatType.Def("infinite?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("math", "IsInf", rcvr.Expr, bst.Int(0)),
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("finite?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(
					bst.Call("math", "IsInf", rcvr.Expr, bst.Int(0)),
					token.EQL,
					ast.NewIdent("false"),
				),
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("abs", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("math", "Abs", rcvr.Expr),
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("between?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(
					bst.Binary(rcvr.Expr, token.GEQ, args[0].Expr),
					token.LAND,
					bst.Binary(rcvr.Expr, token.LEQ, args[1].Expr),
				),
			}
		},
	})
	FloatType.Def("clamp", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Clamp", rcvr.Expr, args[0].Expr, args[1].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	FloatType.Def("nan?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("math", "IsNaN", rcvr.Expr),
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("to_f", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: rcvr.Expr}
		},
	})
	FloatType.Def("truncate", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun:  it.Get("int"),
					Args: []ast.Expr{bst.Call("math", "Trunc", rcvr.Expr)},
				},
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("divmod", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(FloatType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "FloatDivmod", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	FloatType.Def("modulo", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("math", "Mod", rcvr.Expr, args[0].Expr),
				Imports: []string{"math"},
			}
		},
	})
	FloatType.Def("to_r", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return RationalType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "NewRationalFromFloat", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	FloatType.Def("-@", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.UnaryExpr{Op: token.SUB, X: rcvr.Expr},
			}
		},
	})
	FloatType.Def("positive?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.GTR, &ast.BasicLit{Kind: token.FLOAT, Value: "0.0"}),
			}
		},
	})
	FloatType.Def("negative?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.LSS, &ast.BasicLit{Kind: token.FLOAT, Value: "0.0"}),
			}
		},
	})
	FloatType.Def("to_c", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return ComplexType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(nil, "complex", rcvr.Expr, &ast.BasicLit{Kind: token.FLOAT, Value: "0"}),
			}
		},
	})
}

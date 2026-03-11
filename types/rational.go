package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

var stdlibImport = "github.com/redneckbeard/thanos/stdlib"

func init() {
	rationalProto := newProto("Rational", "Numeric", ClassRegistry)

	rationalProto.Def("numerator", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Numerator"),
			}
		},
	})

	rationalProto.Def("denominator", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Denominator"),
			}
		},
	})

	rationalProto.Def("to_f", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "ToF"),
			}
		},
	})

	rationalProto.Def("to_i", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "ToI"),
			}
		},
	})

	rationalProto.Def("to_s", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "ToS"),
			}
		},
	})

	rationalProto.Def("abs", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return RationalType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Abs"),
			}
		},
	})

	// Arithmetic: +, -, *, /
	for _, op := range []struct {
		name, goMethod string
	}{
		{"+", "Add"},
		{"-", "Sub"},
		{"*", "Mul"},
		{"/", "Div"},
	} {
		goMethod := op.goMethod
		rationalProto.Def(op.name, MethodSpec{
			ReturnType: func(r Type, b Type, args []Type) (Type, error) {
				return RationalType, nil
			},
			TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
				return Transform{
					Expr: bst.Call(rcvr.Expr, goMethod, args[0].Expr),
				}
			},
		})
	}

	rationalProto.Def("-@", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return RationalType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Neg"),
			}
		},
	})

	// puts on Rational should use ToS
	rationalProto.Def("to_s_for_puts", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "ToS"),
			}
		},
	})

	RationalType = Rational{rationalProto}
}

type Rational struct {
	*proto
}

func (r Rational) Equals(t2 Type) bool  { _, ok := t2.(Rational); return ok }
func (r Rational) String() string       { return "RationalType" }
func (r Rational) GoType() string       { return "*stdlib.Rational" }
func (r Rational) IsComposite() bool    { return false }
func (r Rational) HasMethod(m string) bool { return r.proto.HasMethod(m, false) }

func (r Rational) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return r.proto.MustResolve(m, false).ReturnType(r, b, args)
}

func (r Rational) BlockArgTypes(m string, args []Type) []Type {
	return r.proto.MustResolve(m, false).blockArgs(r, args)
}

func (r Rational) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return r.proto.MustResolve(m, false).TransformAST(TypeExpr{r, rcvr}, args, blk, it)
}

func (r Rational) Resolve(m string) (MethodSpec, bool) {
	return r.proto.Resolve(m, false)
}

var RationalType Type

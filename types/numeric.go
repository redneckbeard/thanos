package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Numeric struct {
	*proto
}

var NumericType = Numeric{newProto("Numeric", "Object", ClassRegistry)}

var NumericClass = NewClass("Numeric", "Object", NumericType, ClassRegistry)

func (t Numeric) Equals(t2 Type) bool { return t == t2 }
func (t Numeric) String() string      { return "NumericType" }
func (t Numeric) GoType() string      { return "int" }
func (t Numeric) IsComposite() bool   { return false }

func (t Numeric) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.MustResolve(m).ReturnType(t, b, args)
}

//TODO we don't need this in the interface. Instead, the parser or compiler should retrieve the MethodSpec and check for a not-nil blockArgs (which will then need to be exported
func (t Numeric) BlockArgTypes(m string, args []Type) []Type {
	return t.MustResolve(m).blockArgs(t, args)
}

func (t Numeric) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Numeric) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Numeric) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Numeric) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Numeric) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func numericOperatorSpec(tok token.Token, comparison bool) MethodSpec {
	return MethodSpec{
		ReturnType: func(rcvr Type, blockReturnType Type, args []Type) (Type, error) {
			if comparison {
				return BoolType, nil
			}
			if rcvr == FloatType || args[0] == FloatType {
				return FloatType, nil
			}
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			leftExpr, rightExpr := rcvr.Expr, args[0].Expr
			if rcvr.Type == FloatType && args[0].Type == IntType {
				if _, ok := rightExpr.(*ast.BasicLit); !ok {
					rightExpr = bst.Call(nil, "float64", rightExpr)
				}
			} else if rcvr.Type == IntType && args[0].Type == FloatType {
				if _, ok := leftExpr.(*ast.BasicLit); !ok {
					leftExpr = bst.Call(nil, "float64", leftExpr)
				}
			}
			return Transform{
				Expr: bst.Binary(leftExpr, tok, rightExpr),
			}
		},
	}
}

func init() {
	NumericType.Def("+", numericOperatorSpec(token.ADD, false))
	NumericType.Def("-", numericOperatorSpec(token.SUB, false))
	NumericType.Def("*", numericOperatorSpec(token.MUL, false))
	NumericType.Def("/", numericOperatorSpec(token.QUO, false))
	NumericType.Def("%", numericOperatorSpec(token.REM, false))
	NumericType.Def("<", numericOperatorSpec(token.LSS, true))
	NumericType.Def(">", numericOperatorSpec(token.GTR, true))
	NumericType.Def("<=", numericOperatorSpec(token.LEQ, true))
	NumericType.Def(">=", numericOperatorSpec(token.GEQ, true))
	NumericType.Def("==", numericOperatorSpec(token.EQL, true))
	NumericType.Def("!=", numericOperatorSpec(token.NEQ, true))
	NumericType.Def("**", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if r == IntType && args[0] == IntType {
				return IntType, nil
			}
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			leftExpr, rightExpr := rcvr.Expr, args[0].Expr
			if _, ok := leftExpr.(*ast.BasicLit); !ok && rcvr.Type == IntType {
				leftExpr = bst.Call(nil, "float64", leftExpr)
			}
			if _, ok := rightExpr.(*ast.BasicLit); !ok && args[0].Type == IntType {
				rightExpr = bst.Call(nil, "float64", rightExpr)
			}
			expr := bst.Call("math", "Pow", leftExpr, rightExpr)
			if rcvr.Type == IntType && args[0].Type == IntType {
				expr = bst.Call(nil, "int", expr)
			}
			return Transform{
				Expr: expr,
			}
		},
	})

	NumericType.Def("abs", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var arg ast.Expr
			if lit, ok := rcvr.Expr.(*ast.BasicLit); ok || rcvr.Type == FloatType {
				arg = lit
			} else {
				arg = bst.Call(nil, "float64", rcvr.Expr)
			}
			expr := bst.Call("math", "Abs", arg)
			if rcvr.Type == IntType {
				expr = bst.Call(nil, "int", expr)
			}
			return Transform{
				Expr:    expr,
				Imports: []string{"math"},
			}
		},
	})
	NumericType.Def("negative?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.LSS, bst.Int(0)),
			}
		},
	})
	NumericType.Def("positive?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.GTR, bst.Int(0)),
			}
		},
	})
	NumericType.Def("zero?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.EQL, bst.Int(0)),
			}
		},
	})
}

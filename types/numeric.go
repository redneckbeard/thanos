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

func init() {
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

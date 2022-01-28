package types

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Range struct {
	Element  Type
	Instance instance
}

var RangeClass = NewClass("Range", "Object", nil, ClassRegistry)

func NewRange(inner Type) Type {
	return Range{Element: inner, Instance: RangeClass.Instance}
}

func (t Range) Equals(t2 Type) bool { return t == t2 }
func (t Range) String() string      { return fmt.Sprintf("Range(%s)", t.Element) }
func (t Range) GoType() string      { return fmt.Sprintf("*stdlib.Range[%s]", t.Element.GoType()) }
func (t Range) IsComposite() bool   { return true }
func (t Range) Outer() Type         { return Range{} }
func (t Range) Inner() Type         { return t.Element }
func (t Range) ClassName() string   { return "Range" }
func (t Range) IsMultiple() bool    { return false }

func (t Range) HasMethod(m string) bool {
	return t.Instance.HasMethod(m)
}

func (t Range) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.Instance.MustResolve(m).ReturnType(t, b, args)
}

func (t Range) BlockArgTypes(m string, args []Type) []Type {
	return t.Instance.MustResolve(m).blockArgs(t, args)
}

func (t Range) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.Instance.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Range) Resolve(m string) (MethodSpec, bool) {
	return t.Instance.Resolve(m)
}

func (t Range) MustResolve(m string) MethodSpec {
	return t.Instance.MustResolve(m)
}

func init() {
	RangeClass.Instance.Def("===", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			upperTok := token.LSS
			var lower, upper ast.Expr
			if rangeExpr, ok := rcvr.Expr.(*ast.CompositeLit); ok {
				lower, upper = rangeExpr.Elts[0], rangeExpr.Elts[1]
				if rangeExpr.Elts[2].(*ast.Ident).Name == "true" {
					upperTok = token.LEQ
				}
				return Transform{
					Expr: bst.Binary(
						bst.Binary(args[0].Expr, token.GEQ, lower),
						token.LAND,
						bst.Binary(args[0].Expr, upperTok, upper),
					),
				}
			}
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Covers", args[0].Expr),
			}
		},
	})
}

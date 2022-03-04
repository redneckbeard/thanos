package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Range struct {
	*proto
}

var RangeType = Range{newProto("Range", "Object", ClassRegistry)}

var RangeClass = NewClass("Range", "Object", RangeType, ClassRegistry)

func (t Range) Equals(t2 Type) bool { return t == t2 }
func (t Range) String() string      { return "RangeType" }
func (t Range) GoType() string      { return "range" }
func (t Range) IsComposite() bool   { return false }

func (t Range) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Range) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Range) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Range) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Range) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Range) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Range) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
	RangeType.Def("===", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			upperTok := token.LSS
			rangeExpr := rcvr.Expr.(*ast.CompositeLit)
			if rangeExpr.Elts[2].(*ast.Ident).Name == "true" {
				upperTok = token.LEQ
			}
			return Transform{
				Expr: bst.Binary(
					bst.Binary(args[0].Expr, token.GEQ, rangeExpr.Elts[0]),
					token.LAND,
					bst.Binary(args[0].Expr, upperTok, rangeExpr.Elts[1]),
				),
			}
		},
	})
}

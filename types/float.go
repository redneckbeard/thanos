package types

import (
	"go/ast"

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
}

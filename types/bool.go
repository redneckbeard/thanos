package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

type Bool struct {
	*proto
}

var BoolType = Bool{newProto("Boolean", "Object", ClassRegistry)}

var BoolClass = NewClass("Boolean", "Object", BoolType, ClassRegistry)

func (t Bool) String() string    { return "BoolType" }
func (t Bool) GoType() string    { return "bool" }
func (t Bool) IsComposite() bool { return false }

func (t Bool) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Bool) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Bool) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Bool) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Bool) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Bool) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Bool) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func (t Bool) Equals(t2 Type) bool { return t == t2 }

func init() {
}

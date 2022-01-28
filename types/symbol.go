package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

type Symbol struct {
	*proto
}

var SymbolType = Symbol{newProto("Symbol", "Object", ClassRegistry)}

var SymbolClass = NewClass("Symbol", "Object", SymbolType, ClassRegistry)

func (t Symbol) Equals(t2 Type) bool { return t == t2 }
func (t Symbol) String() string      { return "SymbolType" }
func (t Symbol) GoType() string      { return "string" }
func (t Symbol) IsComposite() bool   { return false }

func (t Symbol) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Symbol) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Symbol) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Symbol) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Symbol) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Symbol) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Symbol) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
}

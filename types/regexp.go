package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

type Regexp struct {
	*proto
}

var RegexpType = Regexp{newProto("Regexp", "Object", ClassRegistry)}

var RegexpClass = NewClass("Regexp", "Object", RegexpType, ClassRegistry)

func (t Regexp) Equals(t2 Type) bool { return t == t2 }
func (t Regexp) String() string      { return "RegexpType" }
func (t Regexp) GoType() string      { return "*regexp.Regexp" }
func (t Regexp) IsComposite() bool   { return false }

func (t Regexp) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Regexp) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Regexp) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Regexp) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Regexp) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Regexp) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Regexp) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
	RegexpType.Def("=~", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			// In reality the match operator returns an int, or nil if there's no match. However, in practical
			// use it is relied on for evaluation to a boolean
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "MatchString", args[0].Expr),
			}
		},
	})
}

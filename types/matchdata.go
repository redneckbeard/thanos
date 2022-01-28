package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/stdlib"
)

type matchData struct {
	*proto
}

var MatchDataType = matchData{newProto("MatchData", "Object", ClassRegistry)}

var MatchDataClass = NewClass("MatchData", "Object", MatchDataType, ClassRegistry)

func (t matchData) Equals(t2 Type) bool { return t == t2 }
func (t matchData) String() string      { return "MatchData" }
func (t matchData) GoType() string      { return "*stdlib.matchData" }
func (t matchData) IsComposite() bool   { return false }

func (t matchData) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t matchData) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t matchData) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{Expr: rcvr, Type: t}, args, blk, it)
}

func (t matchData) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t matchData) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t matchData) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t matchData) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
	MatchDataType.GenerateMethods(&stdlib.MatchData{}, "Get", "GetByName")
	MatchDataType.Def("[]", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var methodName string
			if args[0].Type == IntType {
				methodName = "Get"
			} else if args[0].Type == StringType {
				methodName = "GetByName"
			}
			return Transform{
				Expr: bst.Call(rcvr.Expr, methodName, args[0].Expr),
			}
		},
	})
}

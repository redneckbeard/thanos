package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

type errorType struct {
	*proto
}

var RubyErrorType = errorType{newProto("Error", "Object", ClassRegistry)}

func (t errorType) Equals(t2 Type) bool { return t == t2 }
func (t errorType) String() string      { return "Error" }
func (t errorType) GoType() string      { return "error" }
func (t errorType) IsComposite() bool   { return false }

func (t errorType) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t errorType) BlockArgTypes(m string, args []Type) []Type {
	return nil
}

func (t errorType) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t errorType) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t errorType) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func init() {
	RubyErrorType.Def("message", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Error"),
			}
		},
	})
}

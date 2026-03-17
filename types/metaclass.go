package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Metaclass struct {
	*proto
}

var MetaclassType = Metaclass{newProto("Metaclass", "", ClassRegistry)}

func (t Metaclass) Equals(t2 Type) bool { return t == t2 }
func (t Metaclass) String() string      { return "Metaclass" }
func (t Metaclass) GoType() string      { return "stdlib.Metaclass" }
func (t Metaclass) IsComposite() bool   { return false }
func (t Metaclass) IsMultiple() bool    { return false }
func (t Metaclass) ClassName() string   { return "" }

func (t Metaclass) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return MetaclassType.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Metaclass) BlockArgTypes(m string, args []Type) []Type {
	return MetaclassType.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Metaclass) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return MetaclassType.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Metaclass) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Metaclass) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Metaclass) GetMethodSpec(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func init() {
	MetaclassType.Def("name", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Name"),
			}
		},
	})

	MetaclassType.Def("==", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.EQL, args[0].Expr),
			}
		},
	})

	MetaclassType.Def("!=", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.NEQ, args[0].Expr),
			}
		},
	})
}

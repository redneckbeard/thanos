package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

func FprintVerb(t Type) string {
	switch t {
	case StringType:
		return "%s"
	case IntType:
		return "%d"
	case FloatType:
		return "%f"
	case BoolType:
		return "%b"
	case nil:
		return ""
	default:
		return "%v"
	}
}

type String struct {
	*proto
}

var StringType = String{newProto("String", "Object", ClassRegistry)}

var StringClass = NewClass("String", "Object", StringType, ClassRegistry)

func (t String) Equals(t2 Type) bool { return t == t2 }
func (t String) String() string      { return "StringType" }
func (t String) GoType() string      { return "string" }
func (t String) IsComposite() bool   { return false }

func (t String) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t String) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t String) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t String) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t String) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t String) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t String) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
	StringType.Def("match", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return MatchDataType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "NewMatchData", args[0].Expr, rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	StringType.Def("gsub", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if blk != nil {
				panic("Block arguments not yet supported for gsub")
			}
			if _, ok := args[1].Type.(Hash); ok {
				panic("Hash arguments not yet supported for gsub")
			}
			sub := bst.Call("stdlib", "ConvertFromGsub", UnwrapTypeExprs(args)...)
			subVar := it.New("subbed")
			stmt := bst.Define(subVar, bst.Call(args[0].Expr, "ReplaceAllString", rcvr.Expr, sub))
			return Transform{
				Expr:    subVar,
				Stmts:   []ast.Stmt{stmt},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
}

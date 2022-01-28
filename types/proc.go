package types

import (
	"fmt"
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

type Proc struct {
	Args       []Type
	ReturnType Type
	Instance   instance
}

var ProcClass = NewClass("Proc", "Object", nil, ClassRegistry)

func NewProc() *Proc {
	return &Proc{Instance: ProcClass.Instance}
}

func (t *Proc) Equals(t2 Type) bool { return t == t2 }
func (t *Proc) String() string      { return fmt.Sprintf("Proc(%v) -> %s", t.Args, t.ReturnType) }
func (t *Proc) GoType() string      { return "func" }
func (t *Proc) IsComposite() bool   { return false }
func (t *Proc) ClassName() string   { return "Proc" }
func (t *Proc) IsMultiple() bool    { return false }

func (t *Proc) HasMethod(method string) bool {
	return t.Instance.HasMethod(method)
}

func (t *Proc) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.Instance.MustResolve(m).ReturnType(t, b, args)
}

func (t *Proc) BlockArgTypes(m string, args []Type) []Type {
	return t.Instance.MustResolve(m).blockArgs(t, args)
}

func (t *Proc) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.Instance.MustResolve(m).TransformAST(TypeExpr{Expr: rcvr, Type: t}, args, blk, it)
}

func init() {
	ProcClass.Instance.Def("call", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(*Proc).ReturnType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(nil, rcvr.Expr, UnwrapTypeExprs(args)...),
			}
		},
	})
}

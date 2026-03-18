package types

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Optional struct {
	Element  Type
	Instance instance
}

var OptionalClass = NewClass("Optional", "Object", nil, ClassRegistry)

func NewOptional(inner Type) Type {
	return Optional{Element: inner, Instance: OptionalClass.Instance}
}

func (t Optional) Equals(t2 Type) bool {
	if o, ok := t2.(Optional); ok {
		return t.Element.Equals(o.Element)
	}
	return false
}
func (t Optional) String() string    { return fmt.Sprintf("Optional(%s)", t.Element) }
func (t Optional) GoType() string    { return fmt.Sprintf("*%s", t.Element.GoType()) }
func (t Optional) IsComposite() bool { return true }
func (t Optional) Outer() Type       { return Optional{} }
func (t Optional) Inner() Type       { return t.Element }
func (t Optional) ClassName() string { return "Optional" }
func (t Optional) IsMultiple() bool  { return false }

func (t Optional) HasMethod(method string) bool {
	if t.Instance.HasMethod(method) {
		return true
	}
	return t.Element.HasMethod(method)
}

func (t Optional) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	if t.Instance.HasMethod(m) {
		return t.Instance.MustResolve(m).ReturnType(t, b, args)
	}
	return t.Element.MethodReturnType(m, b, args)
}

func (t Optional) GetMethodSpec(m string) (MethodSpec, bool) {
	if spec, ok := t.Instance.Resolve(m); ok {
		return spec, true
	}
	return t.Element.GetMethodSpec(m)
}

func (t Optional) BlockArgTypes(m string, args []Type) []Type {
	if t.Instance.HasMethod(m) {
		return t.Instance.MustResolve(m).blockArgs(t, args)
	}
	return t.Element.BlockArgTypes(m, args)
}

func (t Optional) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	if t.Instance.HasMethod(m) {
		return t.Instance.MustResolve(m).TransformAST(TypeExpr{Expr: rcvr, Type: t}, args, blk, it)
	}
	// Dereference the pointer so the inner type's method sees a value receiver
	deref := &ast.StarExpr{X: rcvr}
	return t.Element.TransformAST(m, deref, args, blk, it)
}

func init() {
	// Logical operators on Optional: compare against nil without dereferencing.
	optLogical := func(tok token.Token) MethodSpec {
		return MethodSpec{
			ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
				return BoolType, nil
			},
			TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
				left := bst.Binary(rcvr.Expr, token.NEQ, it.Get("nil"))
				right := args[0].Expr
				if args[0].Type != BoolType {
					if _, isOpt := args[0].Type.(Optional); isOpt {
						right = bst.Binary(right, token.NEQ, it.Get("nil"))
					}
				}
				return Transform{
					Expr: bst.Binary(left, tok, right),
				}
			},
		}
	}
	OptionalClass.Instance.Def("&&", optLogical(token.LAND))

	OptionalClass.Instance.Def("||", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			opt := r.(Optional)
			// If RHS matches the inner type, result is unwrapped
			if args[0].Equals(opt.Element) {
				return opt.Element, nil
			}
			// Otherwise fall back to Optional
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			opt := rcvr.Type.(Optional)
			if args[0].Type.Equals(opt.Element) {
				return Transform{
					Expr:    bst.Call("stdlib", "OrDefault", rcvr.Expr, args[0].Expr),
					Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
				}
			}
			// Non-matching types: just use != nil || check
			return Transform{
				Expr: bst.Binary(
					bst.Binary(rcvr.Expr, token.NEQ, it.Get("nil")),
					token.LOR,
					bst.Binary(args[0].Expr, token.NEQ, it.Get("nil")),
				),
			}
		},
	})
}

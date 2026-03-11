package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

func init() {
	complexProto := newProto("Complex", "Numeric", ClassRegistry)

	// complex.real → real(c)
	complexProto.Def("real", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{Fun: ast.NewIdent("real"), Args: []ast.Expr{rcvr.Expr}},
			}
		},
	})

	// complex.imaginary → imag(c)
	complexProto.Def("imaginary", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{Fun: ast.NewIdent("imag"), Args: []ast.Expr{rcvr.Expr}},
			}
		},
	})
	complexProto.MakeAlias("imaginary", "imag", false)

	// complex.abs → cmplx.Abs(c)
	complexProto.Def("abs", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("cmplx", "Abs", rcvr.Expr),
				Imports: []string{"math/cmplx"},
			}
		},
	})

	// complex.conjugate → cmplx.Conj(c)
	complexProto.Def("conjugate", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return ComplexType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("cmplx", "Conj", rcvr.Expr),
				Imports: []string{"math/cmplx"},
			}
		},
	})
	complexProto.MakeAlias("conjugate", "conj", false)

	// complex.to_f → real(c)
	complexProto.Def("to_f", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return FloatType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{Fun: ast.NewIdent("real"), Args: []ast.Expr{rcvr.Expr}},
			}
		},
	})

	// complex.to_i → int(real(c))
	complexProto.Def("to_i", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.CallExpr{
					Fun: ast.NewIdent("int"),
					Args: []ast.Expr{
						&ast.CallExpr{Fun: ast.NewIdent("real"), Args: []ast.Expr{rcvr.Expr}},
					},
				},
			}
		},
	})

	ComplexType = Complex{complexProto}
}

type Complex struct {
	*proto
}

func (c Complex) Equals(t2 Type) bool  { _, ok := t2.(Complex); return ok }
func (c Complex) String() string       { return "ComplexType" }
func (c Complex) GoType() string       { return "complex128" }
func (c Complex) IsComposite() bool    { return false }
func (c Complex) HasMethod(m string) bool { return c.proto.HasMethod(m, false) }

func (c Complex) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return c.proto.MustResolve(m, false).ReturnType(c, b, args)
}

func (c Complex) BlockArgTypes(m string, args []Type) []Type {
	return c.proto.MustResolve(m, false).blockArgs(c, args)
}

func (c Complex) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return c.proto.MustResolve(m, false).TransformAST(TypeExpr{c, rcvr}, args, blk, it)
}

func (c Complex) Resolve(m string) (MethodSpec, bool) {
	return c.proto.Resolve(m, false)
}

var ComplexType Type

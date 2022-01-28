package types

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/stdlib"
)

type Set struct {
	Element  Type
	Instance instance
}

var SetClass = NewClass("Set", "Object", nil, ClassRegistry)

func NewSet(inner Type) Type {
	return Set{Element: inner, Instance: SetClass.Instance}
}

func (t Set) Equals(t2 Type) bool { return t == t2 }
func (t Set) String() string      { return fmt.Sprintf("Set{%s}", t.Element) }
func (t Set) GoType() string      { return fmt.Sprintf("*stdlib.Set[%s]", t.Element.GoType()) }
func (t Set) IsComposite() bool   { return true }
func (t Set) Outer() Type         { return Set{} }
func (t Set) Inner() Type         { return t.Element }
func (t Set) ClassName() string   { return "Set" }
func (t Set) IsMultiple() bool    { return false }

func (t Set) HasMethod(m string) bool {
	return t.Instance.HasMethod(m)
}

func (t Set) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.Instance.MustResolve(m).ReturnType(t, b, args)
}

func (t Set) BlockArgTypes(m string, args []Type) []Type {
	return t.Instance.MustResolve(m).blockArgs(t, args)
}

func (t Set) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.Instance.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Set) Resolve(m string) (MethodSpec, bool) {
	return t.Instance.Resolve(m)
}

func (t Set) MustResolve(m string) MethodSpec {
	return t.Instance.MustResolve(m)
}

func init() {
	RegisterType(stdlib.Set[bool]{}, NewSet)
	SetClass.Instance.GenerateMethods(stdlib.Set[bool]{})
	SetClass.Instance.Def("initialize", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if arr, ok := args[0].(Array); ok {
				return NewSet(arr.Element), nil
			}
			return nil, fmt.Errorf("Got %s as argument to set constructor but only Arrays are allowed", args[0])
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "NewSet", UnwrapTypeExprs(args)...),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	SetClass.Def("[]", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("Cannot infer inner type of an empty set")
			}
			argType := args[0]
			for _, arg := range args[1:] {
				if arg != argType {
					return nil, fmt.Errorf("Cannot construct set with heterogenous member types")
				}
			}
			return NewSet(argType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			arr := &ast.CompositeLit{
				Type: &ast.ArrayType{
					Elt: it.Get(args[0].Type.GoType()),
				},
				Elts: UnwrapTypeExprs(args),
			}
			return Transform{
				Expr:    bst.Call("stdlib", "NewSet", arr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	SetClass.Instance.Alias("intersection", "&")
	SetClass.Instance.Alias("union", "|")
	SetClass.Instance.Alias("union", "+")
	SetClass.Instance.Alias("difference", "-")
	SetClass.Instance.Alias("disjoint", "^")
	SetClass.Instance.Alias("superset?", ">=")
	SetClass.Instance.Alias("proper_superset?", ">")
	SetClass.Instance.Alias("subset?", "<=")
	SetClass.Instance.Alias("proper_subset?", "<")
	SetClass.Instance.Def("each", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Set).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var transformedFinal *ast.ExprStmt
			finalStatement := blk.Statements[len(blk.Statements)-1]
			switch f := finalStatement.(type) {
			case *ast.ReturnStmt:
				transformedFinal = &ast.ExprStmt{
					X: f.Results[0],
				}
			case *ast.ExprStmt:
				transformedFinal = f
			default:
				panic("Encountered an unexpected node type")
			}

			blk.Statements[len(blk.Statements)-1] = transformedFinal

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: it.Get("_"),
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})
}

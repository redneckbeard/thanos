package types

import (
	"go/ast"
	"os"
	"testing"
)

type Foo struct{}

func (f *Foo) FuncOne(bar, baz string) int       { return 1 }
func (f *Foo) FuncTwo(bar int, baz bool) float64 { return 1 }

func TestGenerateMethods(t *testing.T) {
	registry := &classRegistry{registry: make(map[string]*Class)}

	foo := newProto("Foo", "", registry)

	registry.Initialize()

	foo.GenerateMethods(&Foo{})
	if !foo.HasMethod("func_one", false) {
		t.Fatal(`foo proto did not get "func_one" defined on it`)
	}
	if !foo.HasMethod("func_two", false) {
		t.Fatal(`foo proto did not get "func_two" defined on it`)
	}

	funcOneSpec := foo.MustResolve("func_one", false)
	retType, _ := funcOneSpec.ReturnType(NilType, NilType, []Type{})
	if retType != IntType {
		t.Fatal("Expected ReturnType method to return IntType")
	}
	transform := funcOneSpec.TransformAST(TypeExpr{}, []TypeExpr{}, nil, nil)
	selector := transform.Expr.(*ast.CallExpr).Fun.(*ast.Ident).Name
	if selector != "FuncOne" {
		t.Fatal("Expected CallExpr to have FuncOne as selector")
	}

	funcTwoSpec := foo.MustResolve("func_two", false)
	retType, _ = funcTwoSpec.ReturnType(NilType, NilType, []Type{})
	if retType != FloatType {
		t.Fatal("Expected ReturnType method to return FloatType")
	}
	transform = funcTwoSpec.TransformAST(TypeExpr{}, []TypeExpr{}, nil, nil)
	selector = transform.Expr.(*ast.CallExpr).Fun.(*ast.Ident).Name
	if selector != "FuncTwo" {
		t.Fatal("Expected CallExpr to have FuncTwo as selector")
	}
}

func TestGenerateMethodsImports(t *testing.T) {
	registry := &classRegistry{registry: make(map[string]*Class)}

	file := newProto("File", "", registry)

	registry.Initialize()

	file.GenerateMethods(&os.File{})

	createSpec := file.MustResolve("name", false)
	transform := createSpec.TransformAST(TypeExpr{}, []TypeExpr{}, nil, nil)

	if len(transform.Imports) < 1 || transform.Imports[0] != "os" {
		t.Fatal("don't have an import", transform.Imports)
	}
}

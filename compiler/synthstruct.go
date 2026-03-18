package compiler

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

// compileSynthStruct emits a Go struct type declaration plus Get and Set
// methods for a synthesized struct derived from a Ruby Tuple.
func (g *GoProgram) compileSynthStruct(ss *types.SynthStruct) []ast.Decl {
	var decls []ast.Decl

	// 1. Struct type declaration
	fields := &ast.FieldList{}
	for _, f := range ss.Fields {
		fields.List = append(fields.List, &ast.Field{
			Names: []*ast.Ident{g.it.Get(f.Name)},
			Type:  g.it.Get(f.Type.GoType()),
		})
	}
	decls = append(decls, &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: g.it.Get(ss.Name),
				Type: &ast.StructType{Fields: fields},
			},
		},
	})

	// 2. Get(i int) interface{} method
	decls = append(decls, g.synthGetMethod(ss))

	// 3. Set(i int, v interface{}) method
	decls = append(decls, g.synthSetMethod(ss))

	return decls
}

func (g *GoProgram) synthGetMethod(ss *types.SynthStruct) ast.Decl {
	rcvrName := "s"

	var cases []ast.Stmt
	for i, f := range ss.Fields {
		cases = append(cases, &ast.CaseClause{
			List: []ast.Expr{g.it.Get(fmt.Sprintf("%d", i))},
			Body: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.SelectorExpr{X: g.it.Get(rcvrName), Sel: g.it.Get(f.Name)},
					},
				},
			},
		})
	}
	cases = append(cases, &ast.CaseClause{
		Body: []ast.Stmt{
			&ast.ExprStmt{X: bst.Call(nil, "panic", &ast.BasicLit{Kind: token.STRING, Value: `"index out of range"`})},
		},
	})

	return &ast.FuncDecl{
		Name: g.it.Get("Get"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{g.it.Get(rcvrName)},
					Type:  &ast.StarExpr{X: g.it.Get(ss.Name)},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{g.it.Get("i")},
						Type:  g.it.Get("int"),
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: g.it.Get("interface{}")},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.SwitchStmt{
					Tag:  g.it.Get("i"),
					Body: &ast.BlockStmt{List: cases},
				},
			},
		},
	}
}

func (g *GoProgram) synthSetMethod(ss *types.SynthStruct) ast.Decl {
	rcvrName := "s"

	var cases []ast.Stmt
	for i, f := range ss.Fields {
		cases = append(cases, &ast.CaseClause{
			List: []ast.Expr{g.it.Get(fmt.Sprintf("%d", i))},
			Body: []ast.Stmt{
				bstAssign(
					&ast.SelectorExpr{X: g.it.Get(rcvrName), Sel: g.it.Get(f.Name)},
					&ast.TypeAssertExpr{
						X:    g.it.Get("v"),
						Type: g.it.Get(f.Type.GoType()),
					},
				),
			},
		})
	}
	cases = append(cases, &ast.CaseClause{
		Body: []ast.Stmt{
			&ast.ExprStmt{X: bst.Call(nil, "panic", &ast.BasicLit{Kind: token.STRING, Value: `"index out of range"`})},
		},
	})

	return &ast.FuncDecl{
		Name: g.it.Get("Set"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{g.it.Get(rcvrName)},
					Type:  &ast.StarExpr{X: g.it.Get(ss.Name)},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{g.it.Get("i")},
						Type:  g.it.Get("int"),
					},
					{
						Names: []*ast.Ident{g.it.Get("v")},
						Type:  g.it.Get("interface{}"),
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.SwitchStmt{
					Tag:  g.it.Get("i"),
					Body: &ast.BlockStmt{List: cases},
				},
			},
		},
	}
}

// bstAssign is a helper to create a simple assignment statement.
func bstAssign(lhs, rhs ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{lhs},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{rhs},
	}
}

// compileSynthStructs emits all synthesized struct declarations.
func (g *GoProgram) compileSynthStructs() []ast.Decl {
	var decls []ast.Decl
	for _, ss := range parser.SynthStructs {
		decls = append(decls, g.compileSynthStruct(ss)...)
	}
	return decls
}

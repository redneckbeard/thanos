package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

func (g *GoProgram) CompileModule(mod *parser.Module) []ast.Decl {
	g.addConstants(mod.Constants)

	var decls []ast.Decl

	for _, cls := range mod.Classes {
		decls = append(decls, g.CompileClass(cls)...)
	}

	return decls
}

func (g *GoProgram) CompileClass(c *parser.Class) []ast.Decl {
	className := globalIdents.Get(c.QualifiedName())
	decls := []ast.Decl{}

	structFields := []*ast.Field{}
	for _, t := range c.IVars(nil) {
		name := t.Name
		if t.Readable && t.Writeable {
			name = strings.Title(name)
		}
		structFields = append(structFields, &ast.Field{
			Names: []*ast.Ident{g.it.Get(name)},
			Type:  g.it.Get(t.Type().GoType()),
		})
	}
	g.addConstants(c.Constants)
	decls = append(decls, &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: className,
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: structFields,
					},
				},
			},
		},
	})
	// hand roll a constructor
	params := []*ast.Field{}
	results := []*ast.Field{
		{
			Type: &ast.StarExpr{
				X: className,
			},
		},
	}

	g.newBlockStmt()

	var (
		initialize *parser.Method
		cls        = c
	)

	for cls != nil && initialize == nil {
		initialize = cls.MethodSet.Methods["initialize"]
		if initialize == nil {
			cls = cls.Parent()
		}
	}

	g.appendToCurrentBlock(bst.Define(g.it.Get("newInstance"),
		&ast.UnaryExpr{
			Op: token.AND,
			X: &ast.CompositeLit{
				Type: className,
			},
		}))

	if initialize != nil {
		params = g.GetFuncParams(initialize.Params)
		var args []ast.Expr
		for _, p := range initialize.Params {
			args = append(args, g.it.Get(p.Name))
		}
		g.appendToCurrentBlock(&ast.ExprStmt{
			X: bst.Call("newInstance", "Initialize", args...),
		})
	}

	signature := &ast.FuncType{
		Params: &ast.FieldList{
			List: params,
		},
		Results: &ast.FieldList{
			List: results,
		},
	}

	g.appendToCurrentBlock(&ast.ReturnStmt{
		Results: []ast.Expr{g.it.Get("newInstance")},
	})

	constructor := &ast.FuncDecl{
		Name: g.it.Get(fmt.Sprintf("New%s", c.QualifiedName())),
		Type: signature,
		Body: g.BlockStack.Peek(),
	}

	decls = append(decls, constructor)

	g.BlockStack.Pop()

	var hasToS bool
	for _, m := range c.Methods(nil) {
		decls = append(decls, g.CompileFunc(m, c)...)
		if m.Name == "to_s" {
			hasToS = true
		}
	}

	if hasToS {
		decls = append(decls, g.stringMethod(c))
	}

	return decls
}

func (g *GoProgram) addConstants(constants []*parser.Constant) {
	for _, constant := range constants {
		switch constant.Val.Type() {
		case types.IntType, types.SymbolType, types.FloatType, types.StringType, types.BoolType:
			g.addConstant(g.it.Get(constant.QualifiedName()), g.CompileExpr(constant.Val))
		default:
			g.addGlobalVar(g.it.Get(constant.QualifiedName()), g.it.Get(constant.Val.Type().GoType()), g.CompileExpr(constant.Val))
		}
	}
}

func (g *GoProgram) stringMethod(cls *parser.Class) ast.Decl {
	signature := &ast.FuncType{
		Results: &ast.FieldList{
			List: g.GetReturnType(types.StringType),
		},
	}

	rcvr := strings.ToLower(cls.Name()[:1])

	return &ast.FuncDecl{
		Type: signature,
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						bst.Call(rcvr, "To_s"),
					},
				},
			},
		},
		Name: g.it.Get("String"),
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{g.it.Get(rcvr)},
					Type: &ast.StarExpr{
						X: g.it.Get(cls.QualifiedName()),
					},
				},
			},
		},
	}
}

package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

func (g *GoProgram) CompileClass(c *parser.Class) []ast.Decl {
	className := globalIdents.Get(c.Name())
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
	for _, constant := range c.Constants {
		switch constant.Val.Type() {
		case types.IntType, types.SymbolType, types.FloatType, types.StringType, types.BoolType:
			g.addConstant(g.it.Get(c.Name()+constant.Name), g.CompileExpr(constant.Val))
		default:
			g.addGlobalVar(g.it.Get(c.Name()+constant.Name), g.it.Get(constant.Val.Type().GoType()), g.CompileExpr(constant.Val))
		}
	}
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
		&ast.Field{
			Type: &ast.StarExpr{
				X: className,
			},
		},
	}

	setStructFields := []ast.Expr{}

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

	if initialize != nil {
		// Currently naive but feeling lazy. What we really want here is the last
		// non-operator assignment to every instance var to be converted to a
		// struct initialization k/v pair, every preceding assignment to them to be
		// locals, and every one after to mutate the field on the already-created
		// struct.
		params = g.GetFuncParams(initialize.Params)
		for _, stmt := range initialize.Body.Statements {
			if assign, ok := stmt.(*parser.AssignmentNode); ok {
				if ivar, isIvar := assign.Left[0].(*parser.IVarNode); isIvar {
					name := ivar.NormalizedVal()
					if ivar.IVar().Readable && ivar.IVar().Writeable {
						name = strings.Title(name)
					}
					setStructFields = append(setStructFields, &ast.KeyValueExpr{
						Key:   g.it.Get(name),
						Value: g.CompileExpr(assign.Right[0]),
					})
				}
			}
		}
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
		Results: []ast.Expr{
			&ast.UnaryExpr{
				Op: token.AND,
				X: &ast.CompositeLit{
					Type: className,
					Elts: setStructFields,
				},
			},
		},
	})

	constructor := &ast.FuncDecl{
		Name: g.it.Get(fmt.Sprintf("New%s", c.Name())),
		Type: signature,
		Body: g.currentBlockStmt(),
	}

	decls = append(decls, constructor)

	g.popBlockStmt()

	for _, m := range c.Methods(nil) {
		if m.Name == "initialize" {
			continue
		}
		decls = append(decls, g.CompileFunc(m, c)...)
	}

	return decls
}

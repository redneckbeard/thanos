package compiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

func (g *GoProgram) CompileFunc(m *parser.Method, c *parser.Class) []ast.Decl {
	decls := []ast.Decl{}

	if m.Block != nil {
		funcType := &ast.FuncType{
			Params: &ast.FieldList{
				List: g.GetFuncParams(m.Block.Params),
			},
			Results: &ast.FieldList{
				List: g.GetReturnType(m.Block.ReturnType),
			},
		}
		typeSpec := &ast.TypeSpec{
			Name: g.it.Get(m.Name + strings.Title(m.Block.Name)),
			Type: funcType,
		}
		decls = append(decls, &ast.GenDecl{
			Tok:   token.TYPE,
			Specs: []ast.Spec{typeSpec},
		})
	}

	if c == nil {
		g.PushState(InFuncDeclaration)
	} else {
		g.currentRcvr = g.it.Get(strings.ToLower(c.Name()[:1]))
		g.PushState(InMethodDeclaration)
	}
	g.ScopeChain = m.Scope
	g.pushTracker()
	defer func() {
		g.PopState()
		g.PopScope()
		g.popTracker()
		g.currentRcvr = nil
	}()
	params := g.GetFuncParams(m.Params)
	if m.Block != nil {
		params = append(params, &ast.Field{
			Names: []*ast.Ident{g.it.Get(m.Block.Name)},
			Type:  g.it.Get(m.Name + strings.Title(m.Block.Name)),
		})
	}

	signature := &ast.FuncType{
		Params: &ast.FieldList{
			List: params,
		},
		Results: &ast.FieldList{
			List: g.GetReturnType(m.ReturnType()),
		},
	}

	decl := &ast.FuncDecl{
		Type: signature,
		Body: g.CompileBlockStmt(m.Body.Statements),
		Name: g.it.Get(m.GoName()),
	}

	if c != nil {
		className := g.it.Get(c.Name())
		decl.Recv = &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Names: []*ast.Ident{g.currentRcvr},
					Type: &ast.StarExpr{
						X: className,
					},
				},
			},
		}
	}

	decls = append(decls, decl)

	return decls
}

func (g *GoProgram) GetFuncParams(rubyParams []*parser.Param) []*ast.Field {
	params := []*ast.Field{}
	for _, p := range rubyParams {
		var (
			lastParam    *ast.Field
			lastSeenType string
		)
		if len(params) > 0 {
			lastParam = params[len(params)-1]
			lastSeenType = lastParam.Type.(*ast.Ident).Name
		}
		if lastParam != nil && lastSeenType == p.Type().GoType() {
			lastParam.Names = append(lastParam.Names, g.it.Get(p.Name))
		} else {
			params = append(params, &ast.Field{
				Names: []*ast.Ident{g.it.Get(p.Name)},
				Type:  g.it.Get(p.Type().GoType()),
			})
		}
	}
	return params
}

func (g *GoProgram) GetReturnType(t types.Type) []*ast.Field {
	fields := []*ast.Field{}
	if t.IsMultiple() {
		multiple := t.(types.Multiple)
		for _, t := range multiple {
			fields = append(fields, g.retTypeField(t))
		}
	} else {
		fields = append(fields, g.retTypeField(t))
	}
	return fields
}

func (g *GoProgram) retTypeField(t types.Type) *ast.Field {
	var retType ast.Expr
	switch r := t.(type) {
	case types.Array:
		retType = &ast.ArrayType{
			Elt: g.it.Get(r.Element.GoType()),
		}
	default:
		if r == types.NilType {
			retType = g.it.Get("")
		} else {
			retType = g.it.Get(r.GoType())
		}
	}
	return &ast.Field{Type: retType}
}

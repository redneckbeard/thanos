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
		g.State.Push(InFuncDeclaration)
	} else {
		g.currentRcvr = g.it.Get(strings.ToLower(c.Name()[:1]))
		g.State.Push(InMethodDeclaration)
	}
	g.ScopeChain = m.Scope
	g.currentMethod = m
	g.pushTracker()
	defer func() {
		g.State.Pop()
		g.popTracker()
		g.currentRcvr = nil
		g.currentMethod = nil
	}()
	params := g.GetFuncParams(m.Params)
	if m.Block != nil {
		params = append(params, &ast.Field{
			Names: []*ast.Ident{g.it.Get(m.Block.Name)},
			Type:  g.it.Get(m.Name + strings.Title(m.Block.Name)),
		})
	}

	retFields := g.GetReturnType(m.ReturnType())
	// For methods that mutate slice params, append those params to the
	// return type so callers can reassign (Go slices are pass-by-value).
	for _, paramIdx := range m.MutatedSliceParams {
		p := m.Params[paramIdx]
		retFields = append(retFields, g.retTypeField(p.Type()))
	}

	signature := &ast.FuncType{
		Params: &ast.FieldList{
			List: params,
		},
		Results: &ast.FieldList{
			List: retFields,
		},
	}

	body := g.CompileBlockStmt(m.Body.Statements)

	// Augment return statements to include mutated slice params
	if len(m.MutatedSliceParams) > 0 {
		g.augmentReturnsWithSliceParams(body, m)
	}

	decl := &ast.FuncDecl{
		Type: signature,
		Body: body,
		Name: g.it.Get(m.GoName()),
	}

	// Add type parameters for generic methods
	if len(m.GenericParams) > 0 {
		decl.Type.TypeParams = g.buildTypeParams(m)
	}

	if c != nil {
		className := g.it.Get(c.Type().GoType())
		decl.Recv = &ast.FieldList{
			List: []*ast.Field{
				{
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

func (g *GoProgram) CompileClassMethod(m *parser.Method, c *parser.Class, prefix ...string) []ast.Decl {
	decls := []ast.Decl{}

	if m.Block != nil {
		blockRetType := m.Block.ReturnType
		// If block return type is unknown, infer from the method's return
		// type (e.g., method returns []int → block returns int).
		if blockRetType == nil && m.ReturnType() != nil {
			if retArr, ok := m.ReturnType().(types.Array); ok && retArr.Element != types.AnyType {
				blockRetType = retArr.Element
			}
		}
		funcType := &ast.FuncType{
			Params: &ast.FieldList{
				List: g.GetFuncParams(m.Block.Params),
			},
			Results: &ast.FieldList{
				List: g.GetReturnType(blockRetType),
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

	g.State.Push(InFuncDeclaration)
	g.ScopeChain = m.Scope
	g.currentMethod = m
	// Ensure block param is in scope for compilation (may have been lost
	// during resetForReanalysis which rebuilds locals from Params only).
	if m.Block != nil {
		if _, found := g.ScopeChain.Get(m.Block.Name); !found {
			g.ScopeChain.Set(m.Block.Name, &parser.RubyLocal{})
			if local, ok := g.ScopeChain.Get(m.Block.Name); ok {
				local.(*parser.RubyLocal).SetType(types.NewProc())
			}
		}
	}
	g.pushTracker()
	defer func() {
		g.State.Pop()
		g.popTracker()
		g.currentMethod = nil
	}()

	params := g.GetFuncParams(m.Params)
	if m.Block != nil {
		params = append(params, &ast.Field{
			Names: []*ast.Ident{g.it.Get(m.Block.Name)},
			Type:  g.it.Get(m.Name + strings.Title(m.Block.Name)),
		})
	}

	retFields := g.GetReturnType(m.ReturnType())
	for _, paramIdx := range m.MutatedSliceParams {
		p := m.Params[paramIdx]
		retFields = append(retFields, g.retTypeField(p.Type()))
	}

	signature := &ast.FuncType{
		Params: &ast.FieldList{
			List: params,
		},
		Results: &ast.FieldList{
			List: retFields,
		},
	}

	owner := ""
	if len(prefix) > 0 {
		owner = prefix[0]
	} else if c != nil {
		owner = c.Name()
	}
	funcName := owner + parser.GoName(m.Name)

	body := g.CompileBlockStmt(m.Body.Statements)
	if len(m.MutatedSliceParams) > 0 {
		g.augmentReturnsWithSliceParams(body, m)
	}

	decl := &ast.FuncDecl{
		Type: signature,
		Body: body,
		Name: g.it.Get(funcName),
	}

	if len(m.GenericParams) > 0 {
		decl.Type.TypeParams = g.buildTypeParams(m)
	}

	decls = append(decls, decl)
	return decls
}

func (g *GoProgram) GetFuncParams(rubyParams []*parser.Param) []*ast.Field {
	params := []*ast.Field{}
	var splat *parser.Param
	for _, p := range rubyParams {
		if p.Kind == parser.Splat {
			splat = p
			continue
		}
		var (
			lastParam    *ast.Field
			lastSeenType string
		)
		pType := p.Type()
		goType := "interface{}"
		if pType != nil {
			goType = pType.GoType()
		}
		if len(params) > 0 {
			lastParam = params[len(params)-1]
			lastSeenType = lastParam.Type.(*ast.Ident).Name
		}
		if lastParam != nil && lastSeenType == goType {
			lastParam.Names = append(lastParam.Names, g.it.Get(p.Name))
		} else {
			params = append(params, &ast.Field{
				Names: []*ast.Ident{g.it.Get(p.Name)},
				Type:  g.it.Get(goType),
			})
		}
	}
	if splat != nil && splat.Type() != nil {
		params = append(params, &ast.Field{
			Names: []*ast.Ident{g.it.Get(splat.Name)},
			Type:  &ast.Ellipsis{Elt: g.it.Get(splat.Type().(types.Array).Inner().GoType())},
		})
	}
	return params
}

// augmentReturnsWithSliceParams walks a function body and appends the
// mutated slice param identifiers to every return statement.
func (g *GoProgram) augmentReturnsWithSliceParams(body *ast.BlockStmt, m *parser.Method) {
	var paramIdents []ast.Expr
	for _, idx := range m.MutatedSliceParams {
		paramIdents = append(paramIdents, g.it.Get(m.Params[idx].Name))
	}
	augmentReturns(body.List, paramIdents)
}

func augmentReturns(stmts []ast.Stmt, extra []ast.Expr) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.ReturnStmt:
			s.Results = append(s.Results, extra...)
		case *ast.IfStmt:
			augmentReturns(s.Body.List, extra)
			if s.Else != nil {
				if block, ok := s.Else.(*ast.BlockStmt); ok {
					augmentReturns(block.List, extra)
				} else if ifStmt, ok := s.Else.(*ast.IfStmt); ok {
					augmentReturns([]ast.Stmt{ifStmt}, extra)
				}
			}
		case *ast.ForStmt:
			augmentReturns(s.Body.List, extra)
		case *ast.RangeStmt:
			augmentReturns(s.Body.List, extra)
		case *ast.BlockStmt:
			augmentReturns(s.List, extra)
		}
	}
}

// buildTypeParams creates the Go type parameter list for a generic method.
func (g *GoProgram) buildTypeParams(m *parser.Method) *ast.FieldList {
	seen := map[string]bool{}
	fields := []*ast.Field{}
	for _, gp := range m.GenericParams {
		if seen[gp.Name] {
			continue
		}
		seen[gp.Name] = true
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{g.it.Get(gp.Name)},
			Type:  g.it.Get(gp.Constraint),
		})
	}
	return &ast.FieldList{List: fields}
}

func (g *GoProgram) GetReturnType(t types.Type) []*ast.Field {
	fields := []*ast.Field{}
	if t == nil || t == types.NilType {
		return fields
	}
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

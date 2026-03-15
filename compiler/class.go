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
	decls := g.addConstants(mod.Constants)

	for _, cls := range mod.Classes {
		decls = append(decls, g.CompileClass(cls)...)
	}

	// Compile module class methods (def self.x) as standalone functions
	for _, m := range mod.ClassMethods {
		decls = append(decls, g.CompileClassMethod(m, nil, mod.Name())...)
	}

	return decls
}

func (g *GoProgram) CompileClass(c *parser.Class) []ast.Decl {
	className := globalIdents.Get(g.localName(c.QualifiedName()))
	decls := []ast.Decl{}

	structFields := []*ast.Field{}
	for _, t := range c.IVars(nil) {
		if t.Type() == nil {
			continue
		}
		name := t.Name
		if t.Readable && t.Writeable {
			name = strings.Title(name)
		}
		structFields = append(structFields, &ast.Field{
			Names: []*ast.Ident{g.it.Get(name)},
			Type:  g.it.Get(t.Type().GoType()),
		})
	}
	decls = append(decls, g.addConstants(c.Constants)...)

	// Emit package-level vars for class variables (@@var)
	for _, cvar := range c.CVars() {
		if cvar.Type() != nil {
			varName := strings.ToLower(c.Name()[:1]) + c.Name()[1:] + strings.Title(cvar.Name)
			g.addGlobalVar(globalIdents.Get(varName), g.it.Get(cvar.Type().GoType()), nil)
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
	if initialize != nil && (initialize.IsUncallable() || !c.IsUsed()) {
		initialize = nil
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
		Name: g.it.Get(fmt.Sprintf("New%s", g.localName(c.QualifiedName()))),
		Type: signature,
		Body: g.BlockStack.Peek(),
	}

	decls = append(decls, constructor)

	g.BlockStack.Pop()

	var hasToS bool
	if c.IsUsed() {
		for _, m := range c.Methods(nil) {
			if m.IsUncallable() {
				continue
			}
			decls = append(decls, g.CompileFunc(m, c)...)
			if m.Name == "to_s" {
				hasToS = true
			}
		}
	}

	// Compile class methods (def self.x) as standalone functions
	for _, m := range c.ClassMethods {
		decls = append(decls, g.CompileClassMethod(m, c)...)
	}

	if hasToS {
		decls = append(decls, g.stringMethod(c))
	}

	for _, alias := range c.Aliases {
		if orig, ok := c.MethodSet.Methods[alias.OldName]; ok {
			decls = append(decls, g.compileAlias(alias, orig, c))
		}
	}

	return decls
}

func (g *GoProgram) cvarGoName(n *parser.CVarNode) string {
	className := strings.ToLower(n.Class.Name()[:1]) + n.Class.Name()[1:]
	return className + strings.Title(n.NormalizedVal())
}

func (g *GoProgram) compileAlias(alias parser.Alias, orig *parser.Method, c *parser.Class) ast.Decl {
	rcvr := strings.ToLower(c.Name()[:1])

	// Build args to forward
	var args []ast.Expr
	params := g.GetFuncParams(orig.Params)
	for _, p := range orig.Params {
		args = append(args, g.it.Get(p.Name))
	}

	call := bst.Call(g.it.Get(rcvr), orig.GoName(), args...)

	var body []ast.Stmt
	if orig.ReturnType() != nil && orig.ReturnType() != types.NilType {
		body = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}
	} else {
		body = []ast.Stmt{&ast.ExprStmt{X: call}}
	}

	goName := parser.GoName(alias.NewName)

	return &ast.FuncDecl{
		Name: g.it.Get(goName),
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: params},
			Results: &ast.FieldList{
				List: g.GetReturnType(orig.ReturnType()),
			},
		},
		Body: &ast.BlockStmt{List: body},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{g.it.Get(rcvr)},
					Type:  &ast.StarExpr{X: g.it.Get(c.Type().GoType())},
				},
			},
		},
	}
}

func (g *GoProgram) addConstants(constants []*parser.Constant) []ast.Decl {
	var initDecls []ast.Decl
	for _, constant := range constants {
		name := g.localName(constant.QualifiedName())
		switch constant.Val.Type() {
		case types.IntType, types.SymbolType, types.FloatType, types.StringType, types.BoolType:
			g.addConstant(g.it.Get(name), g.CompileExpr(constant.Val))
		default:
			// Push a temporary block so CompileExpr can use appendToCurrentBlock
			// for complex expressions (e.g. OrderedMap hashes, method calls).
			tempBlock := &ast.BlockStmt{}
			g.BlockStack.Push(tempBlock)
			val := g.CompileExpr(constant.Val)
			g.BlockStack.Pop()
			if len(tempBlock.List) > 0 {
				// Prepended statements need a function body — emit an init().
				varIdent := g.it.Get(name)
				g.addGlobalVar(varIdent, g.it.Get(constant.Val.Type().GoType()), nil)
				tempBlock.List = append(tempBlock.List, &ast.AssignStmt{
					Lhs: []ast.Expr{varIdent},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{val},
				})
				initDecls = append(initDecls, &ast.FuncDecl{
					Name: ast.NewIdent("init"),
					Type: &ast.FuncType{Params: &ast.FieldList{}},
					Body: tempBlock,
				})
			} else {
				g.addGlobalVar(g.it.Get(name), g.it.Get(constant.Val.Type().GoType()), val)
			}
		}
	}
	return initDecls
}

// compileDuckInterface emits a Go interface type declaration for a synthesized
// duck-type interface. The interface lists methods called on the parameter
// with signatures derived from the first concrete type's analyzed methods.
func (g *GoProgram) compileDuckInterface(iface *types.DuckInterface) []ast.Decl {
	sigs := parser.BuildInterfaceMethodSignatures(iface)
	if len(sigs) == 0 {
		return nil
	}

	methods := &ast.FieldList{}
	for _, sig := range sigs {
		params := g.GetFuncParams(sig.Params)
		results := g.GetReturnType(sig.RetType)

		var funcParams *ast.FieldList
		if len(params) > 0 {
			funcParams = &ast.FieldList{List: params}
		} else {
			funcParams = &ast.FieldList{}
		}

		var funcResults *ast.FieldList
		if len(results) > 0 {
			funcResults = &ast.FieldList{List: results}
		}

		methods.List = append(methods.List, &ast.Field{
			Names: []*ast.Ident{g.it.Get(sig.GoName)},
			Type: &ast.FuncType{
				Params:  funcParams,
				Results: funcResults,
			},
		})
	}

	return []ast.Decl{
		&ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: g.it.Get(iface.Name),
					Type: &ast.InterfaceType{
						Methods: methods,
					},
				},
			},
		},
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
						X: g.it.Get(g.localName(cls.QualifiedName())),
					},
				},
			},
		},
	}
}

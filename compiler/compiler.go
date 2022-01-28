package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os/exec"
	"sort"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
)

type State string

const (
	InFuncDeclaration   State = "InFuncDeclaration"
	InMethodDeclaration State = "InMethodDeclaration"
	InReturnStatement   State = "InReturnStatement"
	InCondAssignment    State = "InCondAssignment"
	InBlockBody         State = "InBlockBody"
)

var globalIdents = make(bst.IdentTracker)

type GoProgram struct {
	State        *parser.Stack[State]
	ScopeChain   parser.ScopeChain
	Imports      map[string]bool
	CurrentLhs   []parser.Node
	BlockStack   *parser.Stack[*ast.BlockStmt]
	GlobalVars   []*ast.ValueSpec
	Constants    []*ast.ValueSpec
	TrackerStack []bst.IdentTracker
	it           bst.IdentTracker
	currentRcvr  *ast.Ident
}

func Compile(p *parser.Root) (string, error) {
	globalIdents = make(bst.IdentTracker)
	g := &GoProgram{State: &parser.Stack[State]{}, Imports: make(map[string]bool), BlockStack: &parser.Stack[*ast.BlockStmt]{}}
	g.pushTracker()

	f := &ast.File{
		Name: ast.NewIdent("main"),
	}

	decls := []ast.Decl{}

	for _, o := range p.Objects {
		if m, ok := o.(*parser.Method); ok {
			decls = append(decls, g.CompileFunc(m, nil)...)
		}
	}

	for _, mod := range p.TopLevelModules {
		decls = append(decls, g.CompileModule(mod)...)
	}

	for _, class := range p.Classes {
		decls = append(decls, g.CompileClass(class)...)
	}

	mainFunc := &ast.FuncDecl{
		Name: ast.NewIdent("main"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
	}

	g.newBlockStmt()
	g.pushTracker()
	for _, stmt := range p.Statements {
		g.CompileStmt(stmt)
	}
	g.popTracker()
	mainFunc.Body = g.BlockStack.Peek()
	g.BlockStack.Pop()

	decls = append(decls, mainFunc)

	importPaths := []string{}

	for imp := range g.Imports {
		importPaths = append(importPaths, imp)
	}

	sort.Strings(importPaths)

	importSpecs := []ast.Spec{}

	for _, path := range importPaths {
		importSpecs = append(importSpecs, &ast.ImportSpec{
			Path: bst.String(path),
		})
	}

	topDecls := []ast.Decl{}

	if len(importSpecs) > 0 {
		topDecls = append(topDecls, &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: importSpecs,
		})
	}

	for _, spec := range g.Constants {
		topDecls = append(topDecls, &ast.GenDecl{
			Tok:   token.CONST,
			Specs: []ast.Spec{spec},
		})
	}

	for _, spec := range g.GlobalVars {
		topDecls = append(topDecls, &ast.GenDecl{
			Tok:   token.VAR,
			Specs: []ast.Spec{spec},
		})
	}

	f.Decls = append(topDecls, decls...)

	var in, out bytes.Buffer
	err := format.Node(&in, token.NewFileSet(), f)
	if err != nil {
		return "", fmt.Errorf("Error converting AST to []byte: %s", err.Error())
	}

	intermediate := in.String()

	cmd := exec.Command("goimports")
	cmd.Stdin = &in
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		fmt.Println(intermediate)
		return intermediate, fmt.Errorf("Error running gofmt: %s", err.Error())
	}

	return out.String(), nil
}

// A Ruby expression will often translate into multiple Go statements, and so
// we need a way to prepend statements prior to where an expression gets
// translated if required. To achieve this, we maintain a stack of
// *ast.BlockStmt that is pushed to and popped from as we work our way down the
// tree. The top of this stack is available for method translating other nodes
// to append to. Because they can append before they complete, they can get
// preceding variable declarations, loops, etc. in place before the expression
// or statement at hand is added.
func (g *GoProgram) newBlockStmt() *ast.BlockStmt {
	blockStmt := &ast.BlockStmt{}
	g.BlockStack.Push(blockStmt)
	return blockStmt
}

func (g *GoProgram) pushTracker() {
	g.it = make(bst.IdentTracker)
	g.TrackerStack = append(g.TrackerStack, g.it)
}

func (g *GoProgram) popTracker() {
	if len(g.TrackerStack) > 0 {
		g.TrackerStack = g.TrackerStack[:len(g.TrackerStack)-1]
		if len(g.TrackerStack) > 0 {
			g.it = g.TrackerStack[len(g.TrackerStack)-1]
		}
	}
}

func (g *GoProgram) appendToCurrentBlock(stmts ...ast.Stmt) {
	currentBlock := g.BlockStack.Peek()
	currentBlock.List = append(currentBlock.List, stmts...)
}

func (g *GoProgram) AddImports(packages ...string) {
	for _, pkg := range packages {
		if _, present := g.Imports[pkg]; !present {
			g.Imports[pkg] = true
		}
	}
}

func (g *GoProgram) addGlobalVar(name *ast.Ident, typeExpr ast.Expr, val ast.Expr) {
	g.GlobalVars = append(g.GlobalVars, &ast.ValueSpec{
		Names:  []*ast.Ident{name},
		Type:   typeExpr,
		Values: []ast.Expr{val},
	})
}

func (g *GoProgram) addConstant(name *ast.Ident, val ast.Expr) {
	g.Constants = append(g.Constants, &ast.ValueSpec{
		Names:  []*ast.Ident{name},
		Values: []ast.Expr{val},
	})
}

func (g *GoProgram) mapToExprs(nodes []parser.Node) []ast.Expr {
	exprs := []ast.Expr{}
	for _, n := range nodes {
		exprs = append(exprs, g.CompileExpr(n))
	}
	return exprs
}

func isSimple(i interface{}) bool {
	switch i.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		return true
	default:
		return false
	}
}

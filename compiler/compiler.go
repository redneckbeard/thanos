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

const (
	InFuncDeclaration   parser.State = "InFuncDeclaration"
	InMethodDeclaration parser.State = "InMethodDeclaration"
	InReturnStatement   parser.State = "InReturnStatement"
	InCondAssignment    parser.State = "InCondAssignment"
	InBlockBody         parser.State = "InBlockBody"
)

var globalIdents = make(bst.IdentTracker)

type GoProgram struct {
	*parser.StateMachine
	Imports      map[string]bool
	CurrentLhs   []parser.Node
	BlockStack   []*ast.BlockStmt
	GlobalVars   []*ast.ValueSpec
	Constants    []*ast.ValueSpec
	TrackerStack []bst.IdentTracker
	it           bst.IdentTracker
	currentRcvr  *ast.Ident
}

func Compile(p *parser.Root) (string, error) {
	globalIdents = make(bst.IdentTracker)
	g := &GoProgram{StateMachine: &parser.StateMachine{}, Imports: make(map[string]bool)}
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
	mainFunc.Body = g.currentBlockStmt()
	g.popBlockStmt()

	decls = append(decls, mainFunc)

	importPaths := []string{}

	for imp, _ := range g.Imports {
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
	g.BlockStack = append(g.BlockStack, blockStmt)
	return blockStmt
}

func (g *GoProgram) popBlockStmt() {
	g.BlockStack = g.BlockStack[:len(g.BlockStack)-1]
}

func (g *GoProgram) currentBlockStmt() *ast.BlockStmt {
	return g.BlockStack[len(g.BlockStack)-1]
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
	currentBlock := g.currentBlockStmt()
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

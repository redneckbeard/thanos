package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

type State string

const (
	InFuncDeclaration   State = "InFuncDeclaration"
	InMethodDeclaration State = "InMethodDeclaration"
	InReturnStatement   State = "InReturnStatement"
	InCondAssignment    State = "InCondAssignment"
	InBlockBody         State = "InBlockBody"
)

var globalIdents = bst.NewIdentTracker()

// fmtPiece is either a literal text segment or an interpolation slot.
type fmtPiece struct {
	text     string   // non-empty for literal text
	arg      ast.Expr // non-nil for interpolation slots
	fallback string   // verb from Ruby type (only for interp slots)
}

// deferredSprintf represents a fmt.Sprintf call whose format verbs need
// to be resolved after all transforms (including Remap) have run.
type deferredSprintf struct {
	fmtLit  *ast.BasicLit    // the format string literal to rewrite
	pieces  []fmtPiece
	delim   string           // " or `
	tracker bst.IdentTracker // the tracker active when this was created
}

type GoProgram struct {
	State           *parser.Stack[State]
	ScopeChain      parser.ScopeChain
	Imports         map[string]bool
	CurrentLhs      []parser.Node
	BlockStack      *parser.Stack[*ast.BlockStmt]
	GlobalVars      []*ast.ValueSpec
	Constants       []*ast.ValueSpec
	TrackerStack    []bst.IdentTracker
	Warnings        []string
	Finalizers      []ast.Stmt
	deferredInterps []deferredSprintf
	it              bst.IdentTracker
	currentRcvr     *ast.Ident
	cs              *commentState
	orderSafeHashes map[string]bool
	modulePrefix    string // non-empty when compiling a module into its own package
	currentMethod   *parser.Method
	suppressDeref   bool // suppress *T dereference during ||= compilation
}

// localName strips the module prefix from a qualified name when compiling
// inside a module package. Handles both Ruby prefixes ("FooBaz" → "Baz")
// and Go package qualifiers ("geometry.Pi" → "Pi").
func (g *GoProgram) localName(qualifiedName string) string {
	if g.modulePrefix != "" {
		// Strip Go package qualifier (e.g., "geometry." from "geometry.Pi")
		pkgDot := strings.ToLower(g.modulePrefix) + "."
		if strings.HasPrefix(qualifiedName, pkgDot) {
			return strings.TrimPrefix(qualifiedName, pkgDot)
		}
		// Strip Ruby module prefix (e.g., "Foo" from "FooBaz")
		return strings.TrimPrefix(qualifiedName, g.modulePrefix)
	}
	return qualifiedName
}

// localizeExpr strips self-package qualifiers from idents in an expression
// when compiling inside a module package.
func (g *GoProgram) localizeExpr(expr ast.Expr) ast.Expr {
	if g.modulePrefix == "" || expr == nil {
		return expr
	}
	pkgDot := strings.ToLower(g.modulePrefix) + "."
	ast.Inspect(expr, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if strings.HasPrefix(ident.Name, pkgDot) {
				ident.Name = strings.TrimPrefix(ident.Name, pkgDot)
			}
		}
		return true
	})
	return expr
}

func (g *GoProgram) Warn(lineNo int, msg string) {
	g.Warnings = append(g.Warnings, fmt.Sprintf("warning: line %d: %s", lineNo, msg))
}

// CompileResult holds the output of a compilation — one or more Go source files.
// For simple programs, only "main.go" is present. When Ruby modules are used,
// each module produces a separate Go package in its own subdirectory.
type CompileResult struct {
	Files map[string]string // relative path -> Go source
}

// MainFile returns the main.go source for backward compatibility.
func (r *CompileResult) MainFile() string {
	return r.Files["main.go"]
}

func Compile(p *parser.Root) (*CompileResult, error) {
	globalIdents = bst.NewIdentTracker()
	g := &GoProgram{State: &parser.Stack[State]{}, ScopeChain: p.ScopeChain, Imports: make(map[string]bool), BlockStack: &parser.Stack[*ast.BlockStmt]{}}
	g.cs = newCommentState(p.Comments)
	g.orderSafeHashes = parser.MarkOrderSafeHashes(p.ScopeChain)
	g.pushTracker()

	// Set PackagePath on module classes so transforms can emit correct imports
	if p.ModulePath == "" {
		p.ModulePath = "tmpmod"
	}
	for _, mod := range p.TopLevelModules {
		setModulePackagePaths(mod, p.ModulePath)
	}

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
		// Modules with content (or sub-modules with content) become their own packages
		if moduleHasContent(mod) {
			continue
		}
		decls = append(decls, g.CompileModule(mod)...)
	}

	for _, class := range p.Classes {
		decls = append(decls, g.CompileClass(class)...)
	}

	// Emit duck-type interface declarations
	for _, iface := range parser.DuckInterfaces {
		decls = append(decls, g.compileDuckInterface(iface)...)
	}

	// Emit package-level vars for global variables ($var)
	for name, t := range parser.GlobalVars() {
		if t != nil {
			g.addGlobalVar(globalIdents.Get(name), g.it.Get(t.GoType()), nil)
		}
	}

	// Reserve lines for package decl, imports, other top-level decls.
	// The exact line numbers will be adjusted at the end.
	// Start the main function body at a high offset to leave room for
	// top-level declarations that are assembled after compilation.
	g.cs.goLine = 1000

	mainFunc := &ast.FuncDecl{
		Name: &ast.Ident{Name: "main", NamePos: g.cs.pos(g.cs.goLine)},
		Type: &ast.FuncType{
			Func:   g.cs.pos(g.cs.goLine),
			Params: &ast.FieldList{},
		},
	}

	g.newBlockStmt()
	g.pushTracker()
	hasComments := len(p.Comments) > 0
	for _, stmt := range p.Statements {
		if hasComments {
			beforeLen := len(g.BlockStack.Peek().List)
			g.CompileStmt(stmt)
			for _, goStmt := range g.BlockStack.Peek().List[beforeLen:] {
				g.cs.stmtLines[goStmt] = stmt.LineNo()
			}
		} else {
			g.CompileStmt(stmt)
		}
	}
	g.popTracker()
	mainFunc.Body = g.BlockStack.Peek()
	if hasComments {
		mainFunc.Body.Lbrace = g.cs.pos(1000)
		g.cs.stampBlockWithComments(mainFunc.Body)
	}
	// Append any finalizer statements
	if len(g.Finalizers) > 0 {
		mainFunc.Body.List = append(mainFunc.Body.List, g.Finalizers...)
	}
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

	g.finalizeDeferredInterps()

	fset := token.NewFileSet()
	if hasComments {
		// Position top-level declarations before the main function body.
		topLine := 2
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "main" {
				continue
			}
			topLine++
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Body != nil {
				fd.Type.Func = g.cs.pos(topLine)
				fd.Name.NamePos = g.cs.pos(topLine)
				fd.Body.Lbrace = g.cs.pos(topLine)
				for _, s := range fd.Body.List {
					topLine++
					setStmtPos(s, g.cs.pos(topLine))
				}
				topLine++
				fd.Body.Rbrace = g.cs.pos(topLine)
			} else if gd, ok := d.(*ast.GenDecl); ok {
				gd.TokPos = g.cs.pos(topLine)
				if len(gd.Specs) > 1 {
					gd.Lparen = g.cs.pos(topLine)
					topLine += len(gd.Specs)
					gd.Rparen = g.cs.pos(topLine)
				}
			}
		}
		f.Package = g.cs.pos(1)
		f.Name = &ast.Ident{Name: "main", NamePos: g.cs.pos(1)}
		f.Comments = g.cs.comments
		fset = g.cs.fset
	}

	mainSrc, err := formatAndImports(f, fset)
	if err != nil {
		return nil, err
	}

	result := &CompileResult{Files: map[string]string{"main.go": mainSrc}}

	// Compile each top-level module (and nested sub-modules) into packages
	for _, mod := range p.TopLevelModules {
		if mod.IsFromGem() {
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Fprintf(os.Stderr, "warning: skipping gem module %s compilation: %v\n", mod.Name(), r)
					}
				}()
				g.compileModulePackages(mod, "", p.ScopeChain, result)
			}()
		} else {
			if err := g.compileModulePackages(mod, "", p.ScopeChain, result); err != nil {
				return nil, err
			}
		}
	}

	for _, w := range g.Warnings {
		fmt.Fprintln(os.Stderr, w)
	}

	return result, nil
}

// setModulePackagePaths recursively sets PackagePath on module types and their classes.
func setModulePackagePaths(mod *parser.Module, parentModPath string) {
	pkgName := strings.ToLower(mod.Name())
	pkgPath := parentModPath + "/" + pkgName
	if mod.Type() != nil {
		if cls, ok := mod.Type().(*types.Class); ok {
			cls.PackagePath = pkgPath
		}
	}
	for _, innerCls := range mod.Classes {
		if innerCls.Type() != nil {
			if cls, ok := innerCls.Type().(*types.Class); ok {
				cls.PackagePath = pkgPath
			}
		}
	}
	for _, sub := range mod.Modules {
		setModulePackagePaths(sub, pkgPath)
	}
}

// moduleHasContent returns true if a module or any of its sub-modules
// have class methods, classes, or other content worth compiling.
func moduleHasContent(mod *parser.Module) bool {
	if len(mod.ClassMethods) > 0 || len(mod.Classes) > 0 {
		return true
	}
	for _, sub := range mod.Modules {
		if moduleHasContent(sub) {
			return true
		}
	}
	return false
}

// compileModulePackages recursively compiles a module and its sub-modules into
// separate Go packages. parentPath is the filesystem path prefix (e.g., "outer" for
// a module nested under Outer). Each module with direct content (class methods or
// classes) gets its own package file.
func (g *GoProgram) compileModulePackages(mod *parser.Module, parentPath string, scope parser.ScopeChain, result *CompileResult) (retErr error) {
	if mod.IsFromGem() {
		defer func() {
			if r := recover(); r != nil {
				var buf [4096]byte
				n := runtime.Stack(buf[:], false)
				fmt.Fprintf(os.Stderr, "warning: gem module %s compilation panic: %v\n%s\n", mod.Name(), r, buf[:n])
				retErr = nil // don't propagate
			}
		}()
	}
	pkgName := strings.ToLower(mod.Name())
	dirPath := pkgName
	if parentPath != "" {
		dirPath = parentPath + "/" + pkgName
	}

	// Compile this module's direct content if it has any
	if len(mod.ClassMethods) > 0 || len(mod.Classes) > 0 {
		modG := &GoProgram{
			State:        &parser.Stack[State]{},
			ScopeChain:   scope,
			Imports:      make(map[string]bool),
			BlockStack:   &parser.Stack[*ast.BlockStmt]{},
			modulePrefix: mod.QualifiedName(),
		}
		modG.pushTracker()

		modDecls := modG.addConstants(mod.Constants)
		for _, cls := range mod.Classes {
			if mod.IsFromGem() {
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmt.Fprintf(os.Stderr, "warning: skipping gem class %s compilation: %v\n", cls.Name(), r)
						}
					}()
					decls := modG.CompileClass(cls)
					for _, d := range decls {
						if validateDecl(d) {
							modDecls = append(modDecls, d)
						} else {
							fmt.Fprintf(os.Stderr, "warning: skipping gem class %s decl (invalid Go AST)\n", cls.Name())
						}
					}
				}()
			} else {
				modDecls = append(modDecls, modG.CompileClass(cls)...)
			}
		}
		for _, m := range mod.ClassMethods {
			if m.FromGem {
				// Only compile gem methods whose return type was successfully inferred
				if m.ReturnType() == nil && !m.IsUncallable() {
					continue
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmt.Fprintf(os.Stderr, "warning: skipping gem method %s.%s (compile panic): %v\n", mod.Name(), m.Name, r)
						}
					}()
					decls := modG.CompileClassMethod(m, nil)
					// Validate each decl by attempting to format it — drop broken AST
					for _, d := range decls {
						if validateDecl(d) {
							modDecls = append(modDecls, d)
						} else {
							fmt.Fprintf(os.Stderr, "warning: skipping gem method %s.%s (invalid Go AST)\n", mod.Name(), m.Name)
						}
					}
				}()
			} else {
				if m.IsUncallable() {
					continue
				}
				modDecls = append(modDecls, modG.CompileClassMethod(m, nil)...)
			}
		}

		if len(modDecls) > 0 {
			modFile := &ast.File{
				Name: ast.NewIdent(pkgName),
			}

			modTopDecls := []ast.Decl{}
			modImportSpecs := []ast.Spec{}
			for imp := range modG.Imports {
				modImportSpecs = append(modImportSpecs, &ast.ImportSpec{Path: bst.String(imp)})
			}
			if len(modImportSpecs) > 0 {
				modTopDecls = append(modTopDecls, &ast.GenDecl{Tok: token.IMPORT, Specs: modImportSpecs})
			}
			for _, spec := range modG.Constants {
				modTopDecls = append(modTopDecls, &ast.GenDecl{Tok: token.CONST, Specs: []ast.Spec{spec}})
			}
			for _, spec := range modG.GlobalVars {
				modTopDecls = append(modTopDecls, &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{spec}})
			}
			modFile.Decls = append(modTopDecls, modDecls...)

			var modSrc string
			var fmtErr error
			if mod.IsFromGem() {
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmtErr = fmt.Errorf("format panic: %v", r)
						}
					}()
					modSrc, fmtErr = formatAndImports(modFile, token.NewFileSet())
				}()
			} else {
				modSrc, fmtErr = formatAndImports(modFile, token.NewFileSet())
			}
			if fmtErr != nil {
				if mod.IsFromGem() {
					fmt.Fprintf(os.Stderr, "warning: gem module %s format error: %v\n", mod.Name(), fmtErr)
				} else {
					return fmt.Errorf("error compiling module %s: %w", mod.Name(), fmtErr)
				}
			} else {
				filePath := dirPath + "/" + pkgName + ".go"
				result.Files[filePath] = modSrc
			}
		}
	}

	// Recurse into sub-modules
	for _, sub := range mod.Modules {
		if moduleHasContent(sub) {
			if sub.IsFromGem() {
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmt.Fprintf(os.Stderr, "warning: skipping gem sub-module %s compilation: %v\n", sub.Name(), r)
						}
					}()
					g.compileModulePackages(sub, dirPath, scope, result)
				}()
			} else {
				if err := g.compileModulePackages(sub, dirPath, scope, result); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// validateDecl checks whether a Go AST declaration can be formatted without
// panicking. Returns false if format.Node panics or errors (e.g., nil *ast.Ident).
func validateDecl(d ast.Decl) (valid bool) {
	defer func() {
		if r := recover(); r != nil {
			valid = false
		}
	}()
	var buf bytes.Buffer
	err := format.Node(&buf, token.NewFileSet(), d)
	return err == nil
}

func formatAndImports(f *ast.File, fset *token.FileSet) (string, error) {
	var in, out bytes.Buffer
	err := format.Node(&in, fset, f)
	if err != nil {
		return "", fmt.Errorf("Error converting AST to []byte: %s", err.Error())
	}

	intermediate := in.String()

	goimportsPath, err := exec.LookPath("goimports")
	if err != nil {
		home, _ := os.UserHomeDir()
		goimportsPath = filepath.Join(home, "go", "bin", "goimports")
	}
	cmd := exec.Command(goimportsPath)
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
	g.it = bst.NewIdentTracker()
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
	if len(stmts) == 0 {
		return
	}
	currentBlock := g.BlockStack.Peek()
	currentBlock.List = append(currentBlock.List, stmts...)
}


func (g *GoProgram) AddImports(packages ...string) {
	for _, pkg := range packages {
		// Skip self-imports when compiling inside a module package
		if g.modulePrefix != "" && strings.HasSuffix(pkg, "/"+strings.ToLower(g.modulePrefix)) {
			continue
		}
		if _, present := g.Imports[pkg]; !present {
			g.Imports[pkg] = true
		}
	}
}

func (g *GoProgram) addGlobalVar(name *ast.Ident, typeExpr ast.Expr, val ast.Expr) {
	spec := &ast.ValueSpec{
		Names: []*ast.Ident{name},
		Type:  typeExpr,
	}
	if val != nil {
		spec.Values = []ast.Expr{val}
	}
	g.GlobalVars = append(g.GlobalVars, spec)
}

func (g *GoProgram) addConstant(name *ast.Ident, val ast.Expr) {
	g.Constants = append(g.Constants, &ast.ValueSpec{
		Names:  []*ast.Ident{name},
		Values: []ast.Expr{val},
	})
}

// isOrderSafe checks whether a hash variable can use native map[K]V
// instead of *stdlib.OrderedMap[K,V].
func (g *GoProgram) isOrderSafe(varName string) bool {
	return g.orderSafeHashes[varName]
}

// hashLhsIsOrderSafe checks if the current LHS variable (during assignment
// compilation) is an order-safe hash.
func (g *GoProgram) hashLhsIsOrderSafe() bool {
	if len(g.CurrentLhs) != 1 {
		return false
	}
	if ident, ok := g.CurrentLhs[0].(*parser.IdentNode); ok {
		return g.isOrderSafe(ident.Val)
	}
	return false
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

package bst

import (
	"fmt"
	"go/ast"
)

// goKeywords maps Go reserved words and predeclared identifiers that would
// cause compilation errors if used as variable/parameter names.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// SanitizeName appends an underscore to Go reserved keywords.
func SanitizeName(name string) string {
	if goKeywords[name] {
		return name + "_"
	}
	return name
}

type IdentTracker struct {
	idents map[string][]*ast.Ident
	types  map[string]string
}

func NewIdentTracker() IdentTracker {
	return IdentTracker{
		idents: make(map[string][]*ast.Ident),
		types:  make(map[string]string),
	}
}

func (it IdentTracker) Get(name string) *ast.Ident {
	if i, ok := it.idents[name]; ok {
		return i[0]
	} else {
		ident := ast.NewIdent(SanitizeName(name))
		it.idents[name] = []*ast.Ident{ident}
		return ident
	}
}

func (it IdentTracker) New(name string) *ast.Ident {
	if i, ok := it.idents[name]; ok {
		incName := fmt.Sprintf("%s%d", SanitizeName(name), len(i))
		inc := ast.NewIdent(incName)
		it.idents[name] = append(it.idents[name], inc)
		it.idents[incName] = []*ast.Ident{inc}
		return inc
	} else {
		return it.Get(name)
	}
}

// Remap creates a new versioned ident for name and makes it the primary,
// so all future Get(name) calls return the new version. Already-emitted
// ast.Ident nodes keep their old name. If goType is non-empty, it records
// the new Go type for this variable.
func (it IdentTracker) Remap(name string, goType ...string) *ast.Ident {
	inc := it.New(name)
	it.idents[name][0] = inc
	if len(goType) > 0 {
		it.types[name] = goType[0]
	}
	return inc
}

// SetType records the Go type string for a variable name.
func (it IdentTracker) SetType(name string, goType string) {
	it.types[name] = goType
}

// GoType returns the recorded Go type for a variable name, or "" if unknown.
func (it IdentTracker) GoType(name string) string {
	return it.types[name]
}

package bst

import (
	"fmt"
	"go/ast"
)

type IdentTracker map[string][]*ast.Ident

func (it IdentTracker) Get(name string) *ast.Ident {
	if i, ok := it[name]; ok {
		return i[0]
	} else {
		ident := ast.NewIdent(name)
		it[name] = []*ast.Ident{ident}
		return ident
	}
}

func (it IdentTracker) New(name string) *ast.Ident {
	if i, ok := it[name]; ok {
		incName := fmt.Sprintf("%s%d", name, len(i))
		inc := ast.NewIdent(incName)
		it[name] = append(i, inc)
		it[incName] = []*ast.Ident{inc}
		return inc
	} else {
		return it.Get(name)
	}
}

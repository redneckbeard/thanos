package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

var PredefinedConstants = map[string]struct {
	Type    Type
	Expr    ast.Expr
	Imports []string
}{
	"ARGV": {
		Type:    NewArray(StringType),
		Expr:    bst.Dot("os", "Args"),
		Imports: []string{"os"},
	},
}

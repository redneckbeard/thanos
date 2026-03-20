package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

// GenericParam represents a Go type parameter (e.g., T in func Foo[T comparable]).
// It is used when a Ruby method is called with the same composite structure but
// different element types (e.g., []int and []string for the same param).
type GenericParam struct {
	Name       string // "T", "T1", etc.
	Constraint string // "comparable", "any"
}

func (g GenericParam) GoType() string    { return g.Name }
func (g GenericParam) ClassName() string { return g.Name }
func (g GenericParam) String() string    { return g.Name + " " + g.Constraint }
func (g GenericParam) IsComposite() bool { return false }
func (g GenericParam) IsMultiple() bool  { return false }

func (g GenericParam) Equals(t2 Type) bool {
	if g2, ok := t2.(GenericParam); ok {
		return g.Name == g2.Name
	}
	return false
}

// GenericParam delegates to the comparable built-in methods.
func (g GenericParam) HasMethod(m string) bool {
	switch m {
	case "==", "!=", "nil?", "class", "to_s", "inspect":
		return true
	}
	return false
}

func (g GenericParam) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	switch m {
	case "==", "!=":
		return BoolType, nil
	case "nil?":
		return BoolType, nil
	case "to_s", "inspect":
		return StringType, nil
	}
	return AnyType, nil
}

func (g GenericParam) GetMethodSpec(m string) (MethodSpec, bool) {
	return MethodSpec{}, false
}

func (g GenericParam) BlockArgTypes(m string, args []Type) []Type {
	return nil
}

func (g GenericParam) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	// Comparable operations compile to native Go operators
	switch m {
	case "==":
		return Transform{Expr: bst.Binary(rcvr, token.EQL, args[0].Expr)}
	case "!=":
		return Transform{Expr: bst.Binary(rcvr, token.NEQ, args[0].Expr)}
	}
	return Transform{Expr: rcvr}
}

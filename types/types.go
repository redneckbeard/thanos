//go:generate stringer -type Simple

// package types provides an interface and many implementations of that
// interface as an abstraction, however leaky, of the union of the Go type
// system and the Ruby Object system.
package types

import (
	"go/ast"
	"reflect"
	"strings"

	"github.com/redneckbeard/thanos/bst"
)

type Type interface {
	BlockArgTypes(string, []Type) []Type
	ClassName() string
	Equals(Type) bool
	GoType() string
	HasMethod(string) bool
	IsComposite() bool
	IsMultiple() bool
	MethodReturnType(string, Type, []Type) (Type, error)
	String() string
	TransformAST(string, ast.Expr, []TypeExpr, *Block, bst.IdentTracker) Transform
	SupportsBrackets(Type) string
}

type TypeExpr struct {
	Type Type
	Expr ast.Expr
}

type Simple int

func (t Simple) GoType() string                   { return typeMap[t] }
func (t Simple) IsComposite() bool                { return false }
func (t Simple) IsMultiple() bool                 { return false }
func (t Simple) ClassName() string                { return "" }
func (t Simple) Equals(t2 Type) bool              { return t == t2 }
func (t Simple) SupportsBrackets(arg Type) string { return "" }

// lies but needed for now
func (t Simple) HasMethod(m string) bool                                      { return false }
func (t Simple) MethodReturnType(m string, b Type, args []Type) (Type, error) { return nil, nil }
func (t Simple) BlockArgTypes(m string, args []Type) []Type                   { return []Type{nil} }
func (t Simple) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return Transform{}
}

const (
	ConstType Simple = iota
	NilType
	FuncType
	AnyType
	ErrorType
)

var typeMap = map[Simple]string{
	ConstType: "const",
	NilType:   "nil",
	FuncType:  "func",
}

var goTypeMap = map[reflect.Kind]Type{
	reflect.Int:     IntType,
	reflect.Float64: FloatType,
	reflect.String:  StringType,
	reflect.Bool:    BoolType,
}

type Multiple []Type

func (t Multiple) GoType() string                   { return "" }
func (t Multiple) IsComposite() bool                { return false }
func (t Multiple) IsMultiple() bool                 { return true }
func (t Multiple) ClassName() string                { return "" }
func (t Multiple) SupportsBrackets(arg Type) string { return "" }
func (t Multiple) String() string {
	segments := []string{}
	for _, s := range t {
		segments = append(segments, s.String())
	}
	return strings.Join(segments, ", ")
}
func (t Multiple) HasMethod(m string) bool                                      { return false }
func (t Multiple) MethodReturnType(m string, b Type, args []Type) (Type, error) { return nil, nil }
func (t Multiple) BlockArgTypes(m string, args []Type) []Type                   { return []Type{nil} }
func (t Multiple) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return Transform{}
}
func (t Multiple) Imports(s string) []string { return []string{} }

func (mt Multiple) Equals(t Type) bool {
	mt2, ok := t.(Multiple)
	if !ok {
		return false
	}
	if len(mt) != len(mt2) {
		return false
	}
	for i, t := range mt {
		if t != mt2[i] {
			return false
		}
	}
	return true
}

type CompositeType interface {
	Type
	Outer() Type
}

type Block struct {
	Args       []ast.Expr
	ReturnType Type
	Statements []ast.Stmt
}

package types

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/redneckbeard/thanos/bst"
)

// Tuple represents a fixed-length heterogeneous collection.
// It arises from array literals with mixed element types (e.g. [name, age]).
// Tuples cannot be used as Go slices — they must be consumed by contexts that
// understand their structure (string formatting, destructuring, etc.).
// Pointer receiver so the type is comparable (by identity) when used in == checks.
type Tuple struct {
	Elements []Type
}

func NewTuple(elements []Type) *Tuple {
	return &Tuple{Elements: elements}
}

func (t *Tuple) GoType() string {
	parts := make([]string, len(t.Elements))
	for i, el := range t.Elements {
		if el != nil {
			parts[i] = el.GoType()
		} else {
			parts[i] = "interface{}"
		}
	}
	return fmt.Sprintf("tuple(%s)", strings.Join(parts, ", "))
}

func (t *Tuple) String() string      { return t.GoType() }
func (t *Tuple) Equals(t2 Type) bool { return false }
func (t *Tuple) IsComposite() bool   { return false }
func (t *Tuple) IsMultiple() bool    { return false }
func (t *Tuple) ClassName() string   { return "Tuple" }
func (t *Tuple) HasMethod(m string) bool {
	return false
}

func (t *Tuple) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return nil, fmt.Errorf("Tuple does not support method '%s'", m)
}

func (t *Tuple) GetMethodSpec(m string) (MethodSpec, bool) {
	return MethodSpec{}, false
}

func (t *Tuple) BlockArgTypes(m string, args []Type) []Type {
	return nil
}

func (t *Tuple) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return Transform{}
}

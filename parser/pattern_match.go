package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/types"
)

// PatternMatchNode represents `case expr; in pattern; ...; end`
type PatternMatchNode struct {
	Value     Node
	InClauses []*InClause
	ElseBody  Statements
	Pos
	_type     types.Type
}

func (n *PatternMatchNode) String() string {
	return fmt.Sprintf("(case/in %s ...)", n.Value)
}
func (n *PatternMatchNode) Type() types.Type     { return n._type }
func (n *PatternMatchNode) SetType(t types.Type) { n._type = t }
func (n *PatternMatchNode) Copy() Node           { return n }

func (n *PatternMatchNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	_, err := GetType(n.Value, locals, class)
	if err != nil {
		return nil, err
	}

	var resultType types.Type
	for _, clause := range n.InClauses {
		// Register pattern bindings in a child scope
		clauseScope := NewScope("pattern")
		valueType := n.Value.Type()
		registerPatternBindings(clause.Pattern, valueType, clauseScope)

		extended := locals.Extend(clauseScope)
		if _, err := clause.Pattern.TargetType(extended, class); err != nil {
			return nil, err
		}
		t, err := GetType(clause.Statements, extended, class)
		if err != nil {
			return nil, err
		}
		// Propagate bindings to outer scope so they're visible after the case
		for name, local := range clauseScope.locals {
			locals.Set(name, local)
		}
		if t != nil && t != types.NilType {
			resultType = t
		}
	}
	if n.ElseBody != nil {
		t, err := GetType(n.ElseBody, locals, class)
		if err != nil {
			return nil, err
		}
		if t != nil && t != types.NilType {
			resultType = t
		}
	}
	if resultType == nil {
		resultType = types.NilType
	}
	return resultType, nil
}

// InClause represents `in pattern then stmts`
type InClause struct {
	Pattern    Node
	Statements Statements
	Pos
	_type      types.Type
}

func (n *InClause) String() string       { return fmt.Sprintf("(in %s %s)", n.Pattern, n.Statements) }
func (n *InClause) Type() types.Type     { return n._type }
func (n *InClause) SetType(t types.Type) { n._type = t }
func (n *InClause) Copy() Node           { return n }

func (n *InClause) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return GetType(n.Statements, locals, class)
}

// ArrayPatternNode represents `[a, b, c]` in a pattern context
type ArrayPatternNode struct {
	Elements []Node // each element is a pattern (IdentNode, ArrayPatternNode, WildcardPatternNode, etc.)
	Pos
	_type    types.Type
}

func (n *ArrayPatternNode) String() string {
	return fmt.Sprintf("[%d elements]", len(n.Elements))
}
func (n *ArrayPatternNode) Type() types.Type     { return n._type }
func (n *ArrayPatternNode) SetType(t types.Type) { n._type = t }
func (n *ArrayPatternNode) Copy() Node           { return n }

func (n *ArrayPatternNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	// Array patterns match against arrays; type comes from context
	return n._type, nil
}

// WildcardPatternNode represents `_` in a pattern context
type WildcardPatternNode struct {
	Pos
	_type  types.Type
}

func (n *WildcardPatternNode) String() string       { return "_" }
func (n *WildcardPatternNode) Type() types.Type     { return n._type }
func (n *WildcardPatternNode) SetType(t types.Type) { n._type = t }
func (n *WildcardPatternNode) Copy() Node           { return n }

func (n *WildcardPatternNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.AnyType, nil
}

// registerPatternBindings walks a pattern and registers any variable bindings
// in the given scope with types inferred from the value being matched.
func registerPatternBindings(pattern Node, valueType types.Type, scope Scope) {
	switch p := pattern.(type) {
	case *IdentNode:
		// Variable capture — infer type from the value being matched
		if p.Val == "_" {
			return
		}
		scope.Set(p.Val, &RubyLocal{_type: valueType})
	case *ArrayPatternNode:
		var elemType types.Type
		if arr, ok := valueType.(types.Array); ok {
			elemType = arr.Element
		}
		for _, elem := range p.Elements {
			registerPatternBindings(elem, elemType, scope)
		}
	case *WildcardPatternNode:
		// No binding for wildcards
	case *ArrayNode:
		// Empty array literal in pattern — no bindings
	}
}

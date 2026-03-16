package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/types"
)

type InfixExpressionNode struct {
	Operator string
	Left     Node
	Right    Node
	Pos
	_type    types.Type
}

func (n *InfixExpressionNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left, n.Operator, n.Right)
}
func (n *InfixExpressionNode) Type() types.Type     { return n._type }
func (n *InfixExpressionNode) SetType(t types.Type) { n._type = t }

func (n *InfixExpressionNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	tl, err := GetType(n.Left, locals, class)
	if err != nil {
		return nil, err
	}
	tr, err := GetType(n.Right, locals, class)
	if err != nil {
		return nil, err
	}
	// When the LHS type is nil or AnyType (e.g., Data.define field with
	// unknown type) and the RHS type is known, infer the LHS type from the
	// RHS. This handles patterns like `action == "+"` where action's type
	// is only discoverable from usage context.
	if (tl == nil || tl == types.AnyType) && tr != nil && tr != types.AnyType {
		n.Left.SetType(tr)
		tl = tr
	}
	// If the operator is a user-defined method (e.g., ==, <=>), register
	// a synthetic call so AnalyzeMethodSet can type the params.
	if ms, ok := classMethodSets[tl]; ok {
		if _, userDefined := ms.Methods[n.Operator]; userDefined {
			syntheticCall := &MethodCall{
				Receiver:   n.Left,
				MethodName: n.Operator,
				Args:       ArgsNode{n.Right},
				Pos: Pos{lineNo: n.lineNo},
			}
			ms.AddCall(syntheticCall)
		}
	}
	if n.HasMethod() {
		if t, err := tl.MethodReturnType(n.Operator, nil, []types.Type{tr}); err != nil {
			return nil, NewParseError(n, err.Error())
		} else {
			// Check if this method can refine variable types (e.g., << on empty arrays)
			if ident, ok := n.Left.(*IdentNode); ok {
				// Try to get the method spec and call RefineVariable if it exists
				if spec, hasSpec := tl.GetMethodSpec(n.Operator); hasSpec && spec.RefineVariable != nil {
					spec.RefineVariable(ident.Val, t, locals)
				}
			}
			// Refine hash value type when mutating a hash-accessed element
			// e.g. h["key"] << "val" refines Hash{K, Array{AnyType}} → Hash{K, Array{String}}
			if ba, ok := n.Left.(*BracketAccessNode); ok {
				if spec, hasSpec := tl.GetMethodSpec(n.Operator); hasSpec && spec.RefineVariable != nil {
					if ident, ok := ba.Composite.(*IdentNode); ok {
						if h, isHash := ba.Composite.Type().(types.Hash); isHash {
							if t != tl {
								refined := types.NewHash(h.Key, t)
								if h.HasDefault {
									refined = types.NewDefaultHash(h.Key, t)
								}
								locals.RefineVariableType(ident.Val, refined)
							}
						}
					}
				}
			}
			return t, nil
		}
	}
	return nil, NewParseError(n, "No method `%s` on type %s", n.Operator, tl)
}

func (n *InfixExpressionNode) Copy() Node {
	return &InfixExpressionNode{n.Operator, n.Left.Copy(), n.Right.Copy(), n.Pos, n._type}
}

func (n *InfixExpressionNode) HasMethod() bool {
	if n.Left.Type() != nil {
		return n.Left.Type().HasMethod(n.Operator)
	}
	return false
}

type NotExpressionNode struct {
	Arg    Node
	Pos
	_type  types.Type
}

func (n *NotExpressionNode) String() string       { return fmt.Sprintf("!%s", n.Arg) }
func (n *NotExpressionNode) Type() types.Type     { return n._type }
func (n *NotExpressionNode) SetType(t types.Type) { n._type = types.BoolType }

func (n *NotExpressionNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if _, err := GetType(n.Arg, locals, class); err != nil {
		return nil, err
	}
	return types.BoolType, nil
}

func (n *NotExpressionNode) Copy() Node {
	return &NotExpressionNode{n.Arg.Copy(), n.Pos, n._type}
}

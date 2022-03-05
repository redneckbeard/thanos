package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/types"
)

type InfixExpressionNode struct {
	Operator string
	Left     Node
	Right    Node
	lineNo   int
	_type    types.Type
}

func (n *InfixExpressionNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left, n.Operator, n.Right)
}
func (n *InfixExpressionNode) Type() types.Type     { return n._type }
func (n *InfixExpressionNode) SetType(t types.Type) { n._type = t }
func (n *InfixExpressionNode) LineNo() int          { return n.lineNo }

func (n *InfixExpressionNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	tl, err := GetType(n.Left, locals, class)
	if err != nil {
		return nil, err
	}
	tr, err := GetType(n.Right, locals, class)
	if err != nil {
		return nil, err
	}
	if n.HasMethod() {
		if t, err := tl.MethodReturnType(n.Operator, nil, []types.Type{tr}); err != nil {
			return nil, NewParseError(n, err.Error())
		} else {
			return t, nil
		}
	}
	return nil, NewParseError(n, "No method `%s` on type %s", n.Operator, tl)
}

func (n *InfixExpressionNode) HasMethod() bool {
	if n.Left.Type() != nil {
		return n.Left.Type().HasMethod(n.Operator)
	}
	return false
}

type NotExpressionNode struct {
	Arg    Node
	lineNo int
	_type  types.Type
}

func (n *NotExpressionNode) String() string       { return fmt.Sprintf("!%s", n.Arg) }
func (n *NotExpressionNode) Type() types.Type     { return n._type }
func (n *NotExpressionNode) SetType(t types.Type) { n._type = types.BoolType }
func (n *NotExpressionNode) LineNo() int          { return n.lineNo }

func (n *NotExpressionNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if _, err := GetType(n.Arg, locals, class); err != nil {
		return nil, err
	}
	return types.BoolType, nil
}

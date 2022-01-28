package parser

import "github.com/redneckbeard/thanos/types"

type NoopNode struct {
	lineNo int
}

func (n *NoopNode) String() string       { return "nil" }
func (n *NoopNode) Type() types.Type     { return types.NilType }
func (n *NoopNode) SetType(t types.Type) {}
func (n *NoopNode) LineNo() int          { return n.lineNo }

func (n *NoopNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
}

func (n *NoopNode) Copy() Node { return n }

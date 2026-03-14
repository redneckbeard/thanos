package parser

import "github.com/redneckbeard/thanos/types"

type NoopNode struct {
	Pos
}

func (n *NoopNode) String() string       { return "nil" }
func (n *NoopNode) Type() types.Type     { return types.NilType }
func (n *NoopNode) SetType(t types.Type) {}

func (n *NoopNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
}

func (n *NoopNode) Copy() Node { return n }

type AliasNode struct {
	NewName string
	OldName string
	Pos
}

func (n *AliasNode) String() string       { return "alias " + n.NewName + " " + n.OldName }
func (n *AliasNode) Type() types.Type     { return nil }
func (n *AliasNode) SetType(t types.Type) {}

func (n *AliasNode) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	if class != nil {
		class.Aliases = append(class.Aliases, Alias{NewName: n.NewName, OldName: n.OldName})
	}
	return types.NilType, nil
}

func (n *AliasNode) Copy() Node { return n }

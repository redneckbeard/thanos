package parser

import "github.com/redneckbeard/thanos/types"

// Placeholder in AST for Kernel method lookups
type KernelNode struct{}

func (n *KernelNode) String() string       { return "Kernel" }
func (n *KernelNode) Type() types.Type     { return types.KernelType }
func (n *KernelNode) SetType(t types.Type) {}
func (n *KernelNode) LineNo() int          { return 0 }

func (n *KernelNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.KernelType, nil
}

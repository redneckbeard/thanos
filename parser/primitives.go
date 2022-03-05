package parser

import "github.com/redneckbeard/thanos/types"

type BooleanNode struct {
	Val    string
	lineNo int
}

func (n *BooleanNode) String() string       { return n.Val }
func (n *BooleanNode) Type() types.Type     { return types.BoolType }
func (n *BooleanNode) SetType(t types.Type) {}
func (n *BooleanNode) LineNo() int          { return n.lineNo }

func (n *BooleanNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.BoolType, nil
}

type IntNode struct {
	Val    string
	lineNo int
}

func (n *IntNode) String() string       { return n.Val }
func (n *IntNode) Type() types.Type     { return types.IntType }
func (n *IntNode) SetType(t types.Type) {}
func (n *IntNode) LineNo() int          { return n.lineNo }

func (n *IntNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.IntType, nil
}

type Float64Node struct {
	Val    string
	lineNo int
}

func (n Float64Node) String() string       { return n.Val }
func (n Float64Node) Type() types.Type     { return types.FloatType }
func (n Float64Node) SetType(t types.Type) {}
func (n *Float64Node) LineNo() int         { return n.lineNo }

func (n Float64Node) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.FloatType, nil
}

type SymbolNode struct {
	Val    string
	lineNo int
}

func (n *SymbolNode) String() string       { return n.Val }
func (n *SymbolNode) Type() types.Type     { return types.SymbolType }
func (n *SymbolNode) SetType(t types.Type) {}
func (n *SymbolNode) LineNo() int          { return n.lineNo }

func (n *SymbolNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.SymbolType, nil
}

type NilNode struct {
	lineNo int
}

func (n *NilNode) String() string       { return "nil" }
func (n *NilNode) Type() types.Type     { return types.NilType }
func (n *NilNode) SetType(t types.Type) {}
func (n *NilNode) LineNo() int          { return n.lineNo }

func (n *NilNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
}

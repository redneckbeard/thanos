package parser

import "github.com/redneckbeard/thanos/types"

type IdentNode struct {
	Val        string
	_type      types.Type
	lineNo     int
	MethodCall *MethodCall
}

func (n *IdentNode) String() string       { return n.Val }
func (n *IdentNode) Type() types.Type     { return n._type }
func (n *IdentNode) SetType(t types.Type) { n._type = t }
func (n *IdentNode) LineNo() int          { return n.lineNo }

func (n *IdentNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	local := locals.ResolveVar(n.Val)
	if local == BadLocal {
		return nil, NewParseError(n, "local variable or method '%s' did not have discoverable type", n.Val)
	}
	if m, ok := local.(*MethodCall); ok {
		n.MethodCall = m
	}
	return local.Type(), nil
}

type GVarNode struct {
	Val    string
	_type  types.Type
	lineNo int
}

func (n *GVarNode) String() string       { return n.Val }
func (n *GVarNode) Type() types.Type     { return n._type }
func (n *GVarNode) SetType(t types.Type) { n._type = t }
func (n *GVarNode) LineNo() int          { return n.lineNo }

func (n *GVarNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return nil, nil
}

type ConstantNode struct {
	Val       string
	Namespace string
	_type     types.Type
	lineNo    int
}

func (n *ConstantNode) String() string       { return n.Val }
func (n *ConstantNode) Type() types.Type     { return n._type }
func (n *ConstantNode) SetType(t types.Type) { n._type = t }
func (n *ConstantNode) LineNo() int          { return n.lineNo }

func (n *ConstantNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if local := locals.ResolveVar(n.Val); local == BadLocal {
		return types.ClassRegistry.Get(n.Val)
	} else {
		if constant, ok := local.(*Constant); ok {
			n.Namespace = constant.Class.Name()
		}
		return local.Type(), nil
	}
}

type SelfNode struct {
	_type  types.Type
	lineNo int
}

func (n *SelfNode) String() string       { return "self" }
func (n *SelfNode) Type() types.Type     { return n._type }
func (n *SelfNode) SetType(t types.Type) { n._type = t }
func (n *SelfNode) LineNo() int          { return n.lineNo }

func (n *SelfNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return nil, nil
}

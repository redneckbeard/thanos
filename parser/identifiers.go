package parser

import (
	"strings"

	"github.com/redneckbeard/thanos/types"
)

type IdentNode struct {
	Val        string
	_type      types.Type
	Pos
	MethodCall *MethodCall
}

func (n *IdentNode) String() string       { return n.Val }
func (n *IdentNode) Type() types.Type     { return n._type }
func (n *IdentNode) SetType(t types.Type) { n._type = t }

func (n *IdentNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	local := locals.ResolveVar(n.Val)
	if local == BadLocal || local.Type() == nil {
		// Fall back to Kernel methods for bare identifiers like `params`
		if types.KernelType.HasMethod(n.Val) {
			retType, err := types.KernelType.MethodReturnType(n.Val, nil, nil)
			if err != nil {
				return nil, err
			}
			// Create a synthetic MethodCall so the compiler invokes the transform
			n.MethodCall = &MethodCall{
				Receiver:   &KernelNode{},
				MethodName: n.Val,
				Pos: Pos{lineNo: n.lineNo},
			}
			n.MethodCall.SetType(retType)
			return retType, nil
		}
		// Also check global methods
		if m, ok := globalMethodSet.Methods[n.Val]; ok {
			if err := m.Analyze(globalMethodSet); err != nil {
				return nil, err
			}
			n.MethodCall = &MethodCall{
				Method:     m,
				MethodName: m.Name,
				_type:      m.ReturnType(),
				Pos: Pos{lineNo: n.lineNo},
			}
			return m.ReturnType(), nil
		}
		if local == BadLocal {
			return nil, NewParseError(n, "local variable or method '%s' did not have discoverable type", n.Val)
		}
		return nil, NewParseError(n, "No type inferred for local variable '%s'", n.Val)
	}
	if m, ok := local.(*MethodCall); ok {
		n.MethodCall = m
	}
	return local.Type(), nil
}

func (n *IdentNode) Copy() Node {
	return &IdentNode{n.Val, n._type, n.Pos, n.MethodCall}
}

// globalVarRegistry tracks all global variables ($var) and their types.
var globalVarRegistry = map[string]types.Type{}

type GVarNode struct {
	Val    string
	_type  types.Type
	Pos
}

func (n *GVarNode) String() string       { return n.Val }
func (n *GVarNode) Type() types.Type     { return n._type }
func (n *GVarNode) SetType(t types.Type) {
	n._type = t
	globalVarRegistry[n.NormalizedVal()] = t
}

func (n *GVarNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	name := n.NormalizedVal()
	if t, ok := globalVarRegistry[name]; ok {
		return t, nil
	}
	// First reference — register with nil type (will be set by assignment)
	globalVarRegistry[name] = nil
	return nil, nil
}

func (n *GVarNode) Copy() Node {
	return n
}

func (n *GVarNode) NormalizedVal() string {
	return strings.TrimLeft(n.Val, "$")
}

// GlobalVars returns all registered global variables with their types.
func GlobalVars() map[string]types.Type {
	return globalVarRegistry
}

// ResetGlobalVars clears the global variable registry (for tests).
func ResetGlobalVars() {
	globalVarRegistry = map[string]types.Type{}
}

type ConstantNode struct {
	Val       string
	Namespace string
	_type     types.Type
	Pos
}

func (n *ConstantNode) String() string       { return n.Val }
func (n *ConstantNode) Type() types.Type     { return n._type }
func (n *ConstantNode) SetType(t types.Type) { n._type = t }

func (n *ConstantNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if local := locals.ResolveVar(n.Val); local == BadLocal {
		if t, err := types.ClassRegistry.Get(n.Val); err != nil {
			if t, ok := types.PredefinedConstants[n.Val]; ok {
				return t.Type, nil
			}
			return nil, err
		} else {
			return t, nil
		}
	} else {
		if constant, ok := local.(*Constant); ok {
			n.Namespace = constant.Namespace.QualifiedName()
		}
		return local.Type(), nil
	}
}

func (n *ConstantNode) Copy() Node {
	// constants being constant, there should never be a need to mutate one
	return n
}

type SelfNode struct {
	_type  types.Type
	Pos
}

func (n *SelfNode) String() string       { return "self" }
func (n *SelfNode) Type() types.Type     { return n._type }
func (n *SelfNode) SetType(t types.Type) { n._type = t }

func (n *SelfNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if class != nil && class.Type() != nil {
		if cls, ok := class.Type().(*types.Class); ok {
			return cls.Instance.(types.Type), nil
		}
	}
	return nil, nil
}

func (n *SelfNode) Copy() Node {
	return &SelfNode{n._type, n.Pos}
}

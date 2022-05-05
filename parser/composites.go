package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type ArrayNode struct {
	Args   ArgsNode
	_type  types.Type
	lineNo int
}

func (n *ArrayNode) String() string       { return fmt.Sprintf("[%s]", n.Args) }
func (n *ArrayNode) Type() types.Type     { return n._type }
func (n *ArrayNode) SetType(t types.Type) { n._type = t }
func (n *ArrayNode) LineNo() int          { return n.lineNo }

func (n *ArrayNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	var inner types.Type
	for _, a := range n.Args {
		ta, _ := GetType(a, locals, class)
		if splat, ok := a.(*SplatNode); ok {
			ta = splat.Type().(types.Array).Element
		}
		if inner != nil && ta != inner {
			return nil, NewParseError(n, "Heterogenous array membership detected adding %s", ta)
		} else {
			inner = ta
		}
	}
	if inner == nil {
		if len(n.Args) == 0 {
			inner = types.AnyType
		} else {
			return nil, NewParseError(n, "No inner array type detected")
		}
	}
	return types.NewArray(inner), nil
}

func (n *ArrayNode) Copy() Node {
	return &ArrayNode{n.Args.Copy().(ArgsNode), n._type, n.lineNo}
}

type KeyValuePair struct {
	Key         Node
	Label       string
	Value       Node
	DoubleSplat bool
	_type       types.Type
	lineNo      int
}

func (n *KeyValuePair) String() string {
	if n.DoubleSplat {
		return fmt.Sprintf("**%s", n.Value)
	}
	return fmt.Sprintf("%s => %s", n.Key, n.Value)
}
func (n *KeyValuePair) Type() types.Type     { return n._type }
func (n *KeyValuePair) SetType(t types.Type) { n._type = n.Value.Type() }
func (n *KeyValuePair) LineNo() int          { return n.lineNo }

func (n *KeyValuePair) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return GetType(n.Value, locals, class)
}

func (n KeyValuePair) Copy() Node {
	kv := &KeyValuePair{
		Label:       n.Label,
		Value:       n.Value.Copy(),
		DoubleSplat: n.DoubleSplat,
		_type:       n._type,
		lineNo:      n.lineNo,
	}
	if n.Key != nil {
		n.Key = n.Key.Copy()
	}
	return kv
}

type HashNode struct {
	Pairs  []*KeyValuePair
	_type  types.Type
	lineNo int
}

func (n *HashNode) String() string {
	return fmt.Sprintf("{%s}", stdlib.Join[*KeyValuePair](n.Pairs, ", "))
}
func (n *HashNode) Type() types.Type     { return n._type }
func (n *HashNode) SetType(t types.Type) { n._type = t }
func (n *HashNode) LineNo() int          { return n.lineNo }

func (n *HashNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	var keyType, valueType types.Type
	for _, kv := range n.Pairs {
		if kv.Label != "" {
			keyType = types.SymbolType
		} else {
			tk, _ := GetType(kv.Key, locals, class)
			if keyType != nil && keyType != tk {
				return nil, fmt.Errorf("Heterogenous hash key membership detected adding %s", tk)
			} else {
				keyType = tk
			}
		}
		tv, _ := GetType(kv.Value, locals, class)
		if valueType != nil && valueType != tv {
			return nil, fmt.Errorf("Heterogenous hash value membership detected adding %s", tv)
		} else {
			valueType = tv
		}
	}
	return types.NewHash(keyType, valueType), nil
}

func (n *HashNode) Copy() Node {
	hash := &HashNode{_type: n._type, lineNo: n.lineNo}
	var pairs []*KeyValuePair
	for _, pair := range n.Pairs {
		pairs = append(pairs, pair.Copy().(*KeyValuePair))
	}
	hash.Pairs = pairs
	return hash
}

func (n *HashNode) Merge(other *HashNode) {
	n.Pairs = append(n.Pairs, other.Pairs...)
}

func (n *HashNode) Delete(key string) {
	for i := len(n.Pairs) - 1; i >= 0; i-- {
		if n.Pairs[i].Label == key {
			n.Pairs = append(n.Pairs[0:i], n.Pairs[i+1:len(n.Pairs)]...)
		}
	}
}

type BracketAssignmentNode struct {
	Composite Node
	Args      ArgsNode
	lineNo    int
	_type     types.Type
}

func (n *BracketAssignmentNode) String() string       { return fmt.Sprintf("%s[%s]", n.Composite, n.Args) }
func (n *BracketAssignmentNode) Type() types.Type     { return n._type }
func (n *BracketAssignmentNode) SetType(t types.Type) { n._type = t }
func (n *BracketAssignmentNode) LineNo() int          { return n.lineNo }

func (n *BracketAssignmentNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return GetType(n.Composite, locals, class)
}

func (n *BracketAssignmentNode) Copy() Node {
	return &BracketAssignmentNode{n.Composite.Copy(), n.Args.Copy().(ArgsNode), n.lineNo, n._type}
}

type BracketAccessNode struct {
	Composite Node
	Args      ArgsNode
	lineNo    int
	_type     types.Type
}

func (n *BracketAccessNode) String() string       { return fmt.Sprintf("%s[%s]", n.Composite, n.Args) }
func (n *BracketAccessNode) Type() types.Type     { return n._type }
func (n *BracketAccessNode) SetType(t types.Type) { n._type = t }
func (n *BracketAccessNode) LineNo() int          { return n.lineNo }

func (n *BracketAccessNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	t, err := GetType(n.Composite, locals, class)
	if err != nil {
		return nil, err
	}
	switch comp := t.(type) {
	case nil:
		return nil, fmt.Errorf("Type not inferred")
	case types.Array:
		if r, ok := n.Args[0].(*RangeNode); ok {
			if _, err = GetType(r, locals, class); err != nil {
				return nil, err
			}
			return t, nil
		}
		return comp.Element, nil
	case types.Hash:
		return comp.Value, nil
	case types.String:
		return types.StringType, nil
	default:
		arg := n.Args[0]
		if _, err := GetType(arg, locals, class); err != nil {
			return nil, err
		}
		if t.HasMethod("[]") {
			if t, err := t.MethodReturnType("[]", nil, []types.Type{arg.Type()}); err != nil {
				return nil, NewParseError(n, err.Error())
			} else {
				return t, nil
			}
		}
		return t, NewParseError(n, "%s is not a supported type for bracket access", t)
	}
}

func (n *BracketAccessNode) Copy() Node {
	return &BracketAccessNode{n.Composite.Copy(), n.Args.Copy().(ArgsNode), n.lineNo, n._type}
}

type SplatNode struct {
	Arg   Node
	_type types.Type
}

func (n *SplatNode) String() string       { return "*" + n.Arg.String() }
func (n *SplatNode) Type() types.Type     { return n._type }
func (n *SplatNode) SetType(t types.Type) { n._type = t }
func (n *SplatNode) LineNo() int          { return n.Arg.LineNo() }

func (n *SplatNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	t, err := GetType(n.Arg, locals, class)
	if err != nil {
		return nil, err
	}
	if _, ok := t.(types.Array); !ok {
		return nil, NewParseError(n, "tried to splat '%s' but is not an array", n.Arg).Terminal()
	}
	return t, nil
}

func (n *SplatNode) Copy() Node {
	return &SplatNode{n.Arg, n._type}
}

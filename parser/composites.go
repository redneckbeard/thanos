package parser

import (
	"fmt"
	"strings"

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
		if inner != nil && ta != inner {
			return nil, NewParseError(n, "Heterogenous array membership detected adding %s", ta)
		} else {
			inner = ta
		}
	}
	if inner == nil {
		inner = types.AnyType
	}
	return types.NewArray(inner), nil
}

type KeyValuePair struct {
	Key    Node
	Label  string
	Value  Node
	_type  types.Type
	lineNo int
}

func (n *KeyValuePair) String() string       { return fmt.Sprintf("%s => %s", n.Key, n.Value) }
func (n *KeyValuePair) Type() types.Type     { return n._type }
func (n *KeyValuePair) SetType(t types.Type) { n._type = n.Value.Type() }
func (n *KeyValuePair) LineNo() int          { return n.lineNo }

func (n *KeyValuePair) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return n.Value.TargetType(locals, class)
}

type HashNode struct {
	Pairs  []*KeyValuePair
	_type  types.Type
	lineNo int
}

func (n *HashNode) String() string {
	segments := []string{}
	for _, kv := range n.Pairs {
		segments = append(segments, kv.String())
	}
	return fmt.Sprintf("{%s}", strings.Join(segments, ", "))
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
	default:
		if method := t.SupportsBrackets(n.Args[0].Type()); method != "" {
			if t, err := t.MethodReturnType(method, nil, []types.Type{n.Args[0].Type()}); err != nil {
				return nil, NewParseError(n, err.Error())
			} else {
				return t, nil
			}
		}
		return t, NewParseError(n, "%s is not a supported type for bracket access", t)
	}
}

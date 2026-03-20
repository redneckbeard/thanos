package parser

import (
	"fmt"
	"strconv"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type ArrayNode struct {
	Args    ArgsNode
	_type   types.Type
	Pos
	isEmpty bool // tracks if this is an empty array needing type inference
}

func (n *ArrayNode) String() string       { return fmt.Sprintf("[%s]", n.Args) }
func (n *ArrayNode) Type() types.Type     { return n._type }
func (n *ArrayNode) SetType(t types.Type) { n._type = t }

func (n *ArrayNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	var inner types.Type
	hasNil := false
	for _, a := range n.Args {
		if _, isNil := a.(*NilNode); isNil {
			hasNil = true
			continue
		}
		ta, _ := GetType(a, locals, class)
		if splat, ok := a.(*SplatNode); ok {
			ta = splat.Type().(types.Array).Element
		}
		if inner != nil && ta != inner {
			// Heterogeneous literal: collect all element types into a Tuple
			elementTypes := make([]types.Type, len(n.Args))
			for i, arg := range n.Args {
				elementTypes[i], _ = GetType(arg, locals, class)
			}
			return types.NewTuple(elementTypes), nil
		} else {
			inner = ta
		}
	}
	if inner == nil {
		if len(n.Args) == 0 {
			// Mark this as an empty array that needs type refinement
			n.MarkAsEmptyArray()
			inner = types.AnyType
		} else if hasNil {
			// All elements are nil — we can't infer a type
			return nil, NewParseError(n, "Array of only nil values — cannot infer element type")
		} else {
			return nil, NewParseError(n, "No inner array type detected")
		}
	}
	if hasNil {
		inner = types.NewOptional(inner)
	}
	return types.NewArray(inner), nil
}

// MarkAsEmptyArray marks this array as empty and needing type inference
func (n *ArrayNode) MarkAsEmptyArray() {
	n.isEmpty = true
}

// IsEmpty returns true if this is an empty array needing type inference
func (n *ArrayNode) IsEmpty() bool {
	return n.isEmpty
}

func (n *ArrayNode) Copy() Node {
	return &ArrayNode{n.Args.Copy().(ArgsNode), n._type, n.Pos, n.isEmpty}
}

type KeyValuePair struct {
	Key         Node
	Label       string
	Value       Node
	DoubleSplat bool
	_type       types.Type
	Pos
}

func (n *KeyValuePair) String() string {
	if n.DoubleSplat {
		return fmt.Sprintf("**%s", n.Value)
	}
	return fmt.Sprintf("%s => %s", n.Key, n.Value)
}
func (n *KeyValuePair) Type() types.Type     { return n._type }
func (n *KeyValuePair) SetType(t types.Type) { n._type = n.Value.Type() }

func (n *KeyValuePair) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return GetType(n.Value, locals, class)
}

func (n KeyValuePair) Copy() Node {
	kv := &KeyValuePair{
		Label:       n.Label,
		Value:       n.Value.Copy(),
		DoubleSplat: n.DoubleSplat,
		_type:       n._type,
		Pos: Pos{lineNo: n.lineNo},
	}
	if n.Key != nil {
		n.Key = n.Key.Copy()
	}
	return kv
}

type HashNode struct {
	Pairs  []*KeyValuePair
	_type  types.Type
	Pos
}

func (n *HashNode) String() string {
	return fmt.Sprintf("{%s}", stdlib.Join[*KeyValuePair](n.Pairs, ", "))
}
func (n *HashNode) Type() types.Type     { return n._type }
func (n *HashNode) SetType(t types.Type) { n._type = t }

func (n *HashNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if len(n.Pairs) == 0 {
		return types.NewHash(types.AnyType, types.AnyType), nil
	}
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
	hash := &HashNode{_type: n._type, Pos: Pos{lineNo: n.lineNo}}
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
	Pos
	_type     types.Type
}

func (n *BracketAssignmentNode) String() string       { return fmt.Sprintf("%s[%s]", n.Composite, n.Args) }
func (n *BracketAssignmentNode) Type() types.Type     { return n._type }
func (n *BracketAssignmentNode) SetType(t types.Type) { n._type = t }

func (n *BracketAssignmentNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	t, err := GetType(n.Composite, locals, class)
	if err != nil {
		return nil, err
	}
	// Ensure all bracket args are typed (needed by compiler for expressions like arr[link[1]])
	for _, arg := range n.Args {
		if _, err := GetType(arg, locals, class); err != nil {
			return nil, err
		}
	}
	if h, ok := t.(types.Hash); ok && h.HasDefault && h.Key == types.AnyType {
		if keyType := n.Args[0].Type(); keyType != nil && keyType != types.AnyType {
			refined := types.NewDefaultHash(keyType, h.Value)
			if ident, ok := n.Composite.(*IdentNode); ok {
				locals.RefineVariableType(ident.Val, refined)
			}
		}
	}
	return t, nil
}

func (n *BracketAssignmentNode) Copy() Node {
	return &BracketAssignmentNode{n.Composite.Copy(), n.Args.Copy().(ArgsNode), n.Pos, n._type}
}

type BracketAccessNode struct {
	Composite Node
	Args      ArgsNode
	Pos
	_type     types.Type
}

func (n *BracketAccessNode) String() string       { return fmt.Sprintf("%s[%s]", n.Composite, n.Args) }
func (n *BracketAccessNode) Type() types.Type     { return n._type }
func (n *BracketAccessNode) SetType(t types.Type) { n._type = t }

func (n *BracketAccessNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	t, err := GetType(n.Composite, locals, class)
	if err != nil {
		return nil, err
	}
	// Ensure all bracket args are typed (needed by compiler for expressions like arr[i - 1])
	for _, arg := range n.Args {
		if _, err := GetType(arg, locals, class); err != nil {
			return nil, err
		}
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
		if comp.HasDefault && comp.Key == types.AnyType {
			// Refine key type based on first access
			if keyType, err := GetType(n.Args[0], locals, class); err == nil && keyType != types.AnyType {
				refined := types.NewDefaultHash(keyType, comp.Value)
				if ident, ok := n.Composite.(*IdentNode); ok {
					locals.RefineVariableType(ident.Val, refined)
					ident.SetType(refined) // Update cached type for downstream refinements
				}
			}
		}
		return comp.Value, nil
	case types.String:
		return types.StringType, nil
	case *types.SynthStruct:
		if len(n.Args) > 0 {
			if _, err := GetType(n.Args[0], locals, class); err != nil {
				return nil, err
			}
			if intNode, ok := n.Args[0].(*IntNode); ok {
				idx, _ := strconv.Atoi(intNode.Val)
				if idx >= 0 && idx < len(comp.Fields) {
					return comp.Fields[idx].Type, nil
				}
			}
		}
		return types.AnyType, nil
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
	return &BracketAccessNode{n.Composite.Copy(), n.Args.Copy().(ArgsNode), n.Pos, n._type}
}

type SplatNode struct {
	Arg   Node
	_type types.Type
}

func (n *SplatNode) String() string       { return "*" + n.Arg.String() }
func (n *SplatNode) Type() types.Type     { return n._type }
func (n *SplatNode) SetType(t types.Type) { n._type = t }
func (n *SplatNode) LineNo() int          { return n.Arg.LineNo() }
func (n *SplatNode) File() string         { return n.Arg.File() }

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

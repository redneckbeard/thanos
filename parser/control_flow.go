package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type Condition struct {
	Condition  Node
	True       Statements
	False      Node
	elseBranch bool
	lineNo     int
	_type      types.Type
}

func (n *Condition) String() string {
	if n.Condition == nil {
		return fmt.Sprintf("(else %s)", n.True)
	}
	if n.False == nil {
		return fmt.Sprintf("(if %s %s)", n.Condition, n.True[0])
	}
	return fmt.Sprintf("(if %s %s %s)", n.Condition, n.True[0], n.False)
}
func (n *Condition) Type() types.Type     { return n._type }
func (n *Condition) SetType(t types.Type) { n._type = t }
func (n *Condition) LineNo() int          { return n.lineNo }

func (n *Condition) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if n.Condition != nil {
		GetType(n.Condition, locals, class)
	}
	t1, err1 := GetType(n.True, locals, class)
	// else clause
	if n.False == nil {
		if n.elseBranch {
			return t1, nil
		}
		return types.NilType, nil
	}
	if t2, err2 := GetType(n.False, locals, class); t1 == t2 && err1 == nil && err2 == nil {
		return t1, nil
	}
	return nil, NewParseError(n.Condition, "Different branches of conditional returned different types: %s", n)
}

func (n *Condition) Copy() Node {
	copy := &Condition{True: n.True.Copy().(Statements), lineNo: n.lineNo}
	if n.False != nil {
		copy.False = n.False.Copy()
	}
	if n.Condition != nil {
		copy.Condition = n.Condition.Copy()
	}
	return copy
}

type CaseNode struct {
	Value             Node
	Whens             []*WhenNode
	RequiresExpansion bool
	_type             types.Type
	lineNo            int
}

func (n *CaseNode) String() string {
	return fmt.Sprintf("(case %s %s)", n.Value, stdlib.Join[*WhenNode](n.Whens, "; "))
}
func (n *CaseNode) Type() types.Type     { return n._type }
func (n *CaseNode) SetType(t types.Type) { n._type = t }
func (n *CaseNode) LineNo() int          { return n.lineNo }

func (n *CaseNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	var (
		t           types.Type
		nilTypeSeen bool
	)

	for _, w := range n.Whens {
		for _, cond := range w.Conditions {
			ct, err := GetType(cond, locals, class)
			if err != nil {
				return nil, err
			}
			if ct.HasMethod("===") {
				n.RequiresExpansion = true
			}
		}
		tw, err := GetType(w, locals, class)
		if err != nil {
			return nil, err
		}

		if tw != nil {
			if tw != types.NilType {
				if t != nil && t != tw {
					return nil, NewParseError(w, "Case statement branches return conflicting types %s and %s", t, tw)
				}
				t = tw
			} else {
				nilTypeSeen = true
			}
		}
	}
	if t == nil && nilTypeSeen {
		t = types.NilType
	}
	return t, nil
}

func (n *CaseNode) Copy() Node {
	caseNode := &CaseNode{Value: n.Value.Copy(), RequiresExpansion: n.RequiresExpansion, _type: n._type, lineNo: n.lineNo}
	for _, when := range n.Whens {
		caseNode.Whens = append(caseNode.Whens, when.Copy().(*WhenNode))
	}
	return caseNode
}

type WhenNode struct {
	Conditions ArgsNode
	Statements Statements
	_type      types.Type
	lineNo     int
}

func (n *WhenNode) String() string {
	if n.Conditions == nil {
		return fmt.Sprintf("(else %s)", n.Statements)
	}
	return fmt.Sprintf("(when (%s) %s)", n.Conditions, n.Statements)
}
func (n *WhenNode) Type() types.Type     { return n._type }
func (n *WhenNode) SetType(t types.Type) { n._type = t }
func (n *WhenNode) LineNo() int          { return n.lineNo }

func (n *WhenNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return GetType(n.Statements, locals, class)
}

func (n *WhenNode) Copy() Node {
	return &WhenNode{n.Conditions.Copy().(ArgsNode), n.Statements.Copy().(Statements), n._type, n.lineNo}
}

type WhileNode struct {
	Condition Node
	Body      Statements
	lineNo    int
}

func (n *WhileNode) String() string {
	return fmt.Sprintf("(while %s (%s))", n.Condition, n.Body)
}
func (n *WhileNode) Type() types.Type     { return n.Body.Type() }
func (n *WhileNode) SetType(t types.Type) {}
func (n *WhileNode) LineNo() int          { return n.lineNo }

func (n *WhileNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if _, err := GetType(n.Condition, locals, class); err != nil {
		return nil, err
	}
	if _, err := GetType(n.Body, locals, class); err != nil {
		return nil, err
	}
	return types.NilType, nil
}

func (n *WhileNode) Copy() Node {
	return &WhileNode{n.Condition.Copy(), n.Body.Copy().(Statements), n.lineNo}
}

type ForInNode struct {
	For    []Node
	In     Node
	Body   Statements
	lineNo int
}

func (n *ForInNode) String() string {
	return fmt.Sprintf("(for %s in %s (%s))", n.For, n.In, n.Body)
}
func (n *ForInNode) Type() types.Type     { return n.Body.Type() }
func (n *ForInNode) SetType(t types.Type) {}
func (n *ForInNode) LineNo() int          { return n.lineNo }

func (n *ForInNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	var inType types.Type
	inType, err := GetType(n.In, locals, class)
	if err != nil {
		return nil, err
	}
	if !inType.IsComposite() {
		return nil, NewParseError(n, "For loops over %s not supported", inType)
	}
	if t, ok := inType.(types.Hash); ok {
		if len(n.For) != 2 {
			return nil, NewParseError(n, "For loops over hashes must unpack one key and one value")
		}
		for i, v := range n.For {
			ident, ok := v.(*IdentNode)
			if !ok {
				return nil, NewParseError(n, "Not sure how this even successfully parsed")
			}
			if i == 0 {
				locals.Set(ident.Val, &RubyLocal{_type: t.Key})
			} else {
				locals.Set(ident.Val, &RubyLocal{_type: t.Value})
			}
		}
	} else {
		if len(n.For) != 1 {
			return nil, NewParseError(n, "Destructuring subarrays in for loops not supported")
		}
		ident, ok := n.For[0].(*IdentNode)
		if !ok {
			return nil, NewParseError(n, "Not sure how this even successfully parsed")
		}
		locals.Set(ident.Val, &RubyLocal{_type: inType.(types.CompositeType).Inner()})
	}
	for _, v := range n.For {
		GetType(v, locals, class)
	}
	if _, err := GetType(n.Body, locals, class); err != nil {
		return nil, err
	}
	return inType, nil
}

func (n *ForInNode) Copy() Node {
	return &ForInNode{n.For, n.In, n.Body, n.lineNo}
}

type BreakNode struct {
	lineNo int
}

func (n *BreakNode) String() string {
	return fmt.Sprintf("(break)")
}
func (n *BreakNode) Type() types.Type     { return types.NilType }
func (n *BreakNode) SetType(t types.Type) {}
func (n *BreakNode) LineNo() int          { return n.lineNo }

func (n *BreakNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
}

func (n *BreakNode) Copy() Node {
	return n
}

type NextNode struct {
	lineNo int
}

func (n *NextNode) String() string {
	return fmt.Sprintf("(next)")
}
func (n *NextNode) Type() types.Type     { return types.NilType }
func (n *NextNode) SetType(t types.Type) {}
func (n *NextNode) LineNo() int          { return n.lineNo }

func (n *NextNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
}

func (n *NextNode) Copy() Node {
	return n
}

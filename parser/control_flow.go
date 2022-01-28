package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type Condition struct {
	Condition Node
	True      Statements
	False     Node
	lineNo    int
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
func (n *Condition) Type() types.Type     { return n.True.Type() }
func (n *Condition) SetType(t types.Type) {}
func (n *Condition) LineNo() int          { return n.lineNo }

func (n *Condition) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if n.Condition != nil {
		GetType(n.Condition, locals, class)
	}
	t1, err1 := GetType(n.True, locals, class)
	// else clause
	if n.False == nil {
		return t1, nil
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
	return types.NilType, nil
}

func (n *WhileNode) Copy() Node {
	return &WhileNode{n.Condition.Copy(), n.Body.Copy().(Statements), n.lineNo}
}

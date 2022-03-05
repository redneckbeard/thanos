package parser

import (
	"fmt"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

const (
	Main = "__main__"
)

type Root struct {
	*StateMachine
	Objects          []Node
	Statements       []Node
	Classes          []*Class
	MethodSets       []*MethodSet
	Errors           []error
	ExplicitReturns  []*ReturnNode
	stringStack      []*StringNode
	Comments         map[int]Comment
	currentClass     *Class
	currentMethod    *Method
	inPrivateMethods bool
}

func NewRoot() *Root {
	globalMethodSet = NewMethodSet()
	p := &Root{
		StateMachine: &StateMachine{},
		MethodSets:   []*MethodSet{globalMethodSet},
		Comments:     make(map[int]Comment),
	}
	p.PushScope(NewScope(Main))
	types.ClassRegistry.Initialize()
	return p
}

func (r *Root) AddComment(c Comment) {
	r.Comments[c.LineNo] = c
}

func (r *Root) AddReturn(n *ReturnNode) {
	r.ExplicitReturns = append(r.ExplicitReturns, n)
}

func (r *Root) AddError(err error) {
	r.Errors = append(r.Errors, err)
}

type ParseError struct {
	node Node
	msg  string
}

func (p *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", p.node.LineNo(), p.msg)
}

func NewParseError(node Node, fmtString string, args ...interface{}) error {
	return &ParseError{
		node: node,
		msg:  fmt.Sprintf(fmtString, args...),
	}
}

func (r *Root) CurrentString() *StringNode {
	return r.stringStack[len(r.stringStack)-1]
}

func (r *Root) PushString() {
	r.stringStack = append(r.stringStack, &StringNode{Interps: make(map[int][]Node)})
}

func (r *Root) PopString() *StringNode {
	last := r.stringStack[len(r.stringStack)-1]
	r.stringStack = r.stringStack[:len(r.stringStack)-1]
	return last
}

func (r *Root) PushClass(name string, lineNo int) {
	r.PushState(InClassBody)
	cls := &Class{name: name, lineNo: lineNo, ivars: make(map[string]*IVar)}
	ms := NewMethodSet()
	cls.MethodSet = ms
	ms.Class = cls
	r.currentClass = cls
	r.MethodSets = append(r.MethodSets, ms)
}

func (r *Root) PopClass() *Class {
	class := r.currentClass
	r.MethodSets = r.MethodSets[:len(r.MethodSets)-1]
	r.currentClass = nil
	r.Classes = append(r.Classes, class)
	r.inPrivateMethods = false
	t := class.BuildType()
	classMethodSets[t.Instance.(types.Type)] = class.MethodSet
	r.PopState()
	r.ScopeChain.Set(class.Name(), class)
	return class
}

func (r *Root) ParseError() error {
	if len(r.Errors) > 0 {
		return r.Errors[0]
	}
	return nil
}

func (r *Root) CurrentMethodSet() *MethodSet {
	return r.MethodSets[len(r.MethodSets)-1]
}

func (r *Root) AddMethod(m *Method) {
	m.Body.ExplicitReturns = r.ExplicitReturns
	r.ExplicitReturns = []*ReturnNode{}
	ms := r.CurrentMethodSet()
	ms.Methods[m.Name] = m
	ms.Order = append(ms.Order, m.Name)
	r.currentMethod = nil
}

func (r *Root) GetMethod(name string) (*Method, bool) {
	if method, ok := r.CurrentMethodSet().Methods[name]; ok {
		return method, true
	} else {
		return nil, false
	}
}

func (r *Root) AddCall(c *MethodCall) {
	if c.Receiver != nil {
		switch rcvr := c.Receiver.(type) {
		case *IdentNode:
			loc := r.ScopeChain.ResolveVar(rcvr.Val)
			if loc != BadLocal {
				loc.(*RubyLocal).AddCall(c)
			} else {
				uncalled := &RubyLocal{}
				uncalled.AddCall(c)
				r.ScopeChain.Set(rcvr.Val, uncalled)
				return
			}
		}
	} else if method, found := r.CurrentMethodSet().Methods[c.MethodName]; found {
		if err := method.AnalyzeArguments(r.CurrentMethodSet().Class, c); err != nil {
			r.AddError(err)
			return
		}
	}
	if calls, ok := r.CurrentMethodSet().Calls[c.MethodName]; ok {
		r.CurrentMethodSet().Calls[c.MethodName] = append(calls, c)
	} else {
		r.CurrentMethodSet().Calls[c.MethodName] = []*MethodCall{c}
	}
}

func (r *Root) AddStatement(n Node) {
	switch n.(type) {
	case *Method:
		r.Objects = append(r.Objects, n)
	case *Class:
		// do nothing, handled differently
	default:
		r.Statements = append(r.Statements, n)
	}
}

func (r *Root) AnalyzeMethodSet(ms *MethodSet, rcvr types.Type) error {
	for _, name := range ms.Order {
		m := ms.Methods[name]
		if err := m.Analyze(ms); err != nil {
			return err
		}
	}
	if initialize, ok := ms.Methods["initialize"]; ok {
		if err := initialize.Analyze(ms); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) Analyze() error {
	if len(r.Errors) > 0 {
		return r.Errors[0]
	}
	// First pass, just to pick up method calls
	if len(r.Statements) > 0 {
		if err := (&Body{Statements: r.Statements}).InferReturnType(r.ScopeChain, nil); err != nil {
			// probably this is too aggressive
			return err
		}
	}

	for i := len(r.Classes) - 1; i >= 0; i-- {
		class := r.Classes[i]
		if err := r.AnalyzeMethodSet(class.MethodSet, class.Type()); err != nil {
			return err
		}
	}
	if err := r.AnalyzeMethodSet(r.CurrentMethodSet(), nil); err != nil {
		return err
	}

	for _, calls := range r.CurrentMethodSet().Calls {
		for _, c := range calls {
			GetType(c, r.ScopeChain, r.CurrentMethodSet().Class)
		}
	}
	return nil
}

func (n *Root) String() string {
	tlos := []Node{}
	if n.Objects != nil {
		tlos = append(tlos, n.Objects...)
	}
	if n.Classes != nil {
		for _, cls := range n.Classes {
			tlos = append(tlos, cls)
		}
	}
	if n.Statements != nil {
		tlos = append(tlos, n.Statements...)
	}
	stmts := []string{}
	for _, tlo := range tlos {
		stmts = append(stmts, tlo.String())
	}
	return strings.Join(stmts, "\n")
}

type Comment struct {
	Text   string
	LineNo int
}

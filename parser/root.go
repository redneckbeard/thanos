package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

const (
	Main = "__main__"
)

type State string

const (
	TopLevelStatement  State = "TopLevelStatement"
	InClassBody        State = "InClassBody"
	InModuleBody       State = "InModuleBody"
	InMethodDefinition State = "InMethodDefinition"
	InString           State = "InString"
)

type Root struct {
	State            *Stack[State]
	ScopeChain       ScopeChain
	Objects          []Node
	Statements       []Node
	Classes          []*Class
	MethodSetStack   *Stack[*MethodSet]
	Errors           []error
	ExplicitReturns  []*ReturnNode
	StringStack      *Stack[*StringNode]
	Comments         map[int]Comment
	moduleStack      *Stack[*Module]
	TopLevelModules  []*Module
	currentClass     *Class
	currentMethod    *Method
	inPrivateMethods bool
	nextConstantType int
}

func NewRoot() *Root {
	globalMethodSet = NewMethodSet()
	p := &Root{
		State:          &Stack[State]{},
		StringStack:    &Stack[*StringNode]{},
		moduleStack:    &Stack[*Module]{},
		MethodSetStack: &Stack[*MethodSet]{stack: []*MethodSet{globalMethodSet}},
		Comments:       make(map[int]Comment),
		ScopeChain:     ScopeChain{NewScope(Main)},
	}
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
	node     Node
	msg      string
	terminal bool
}

func (p *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", p.node.LineNo(), p.msg)
}

func (p *ParseError) Terminal() *ParseError {
	p.terminal = true
	return p
}

func NewParseError(node Node, fmtString string, args ...interface{}) *ParseError {
	return &ParseError{
		node: node,
		msg:  fmt.Sprintf(fmtString, args...),
	}
}

func (r *Root) PushModule(name string, lineNo int) {
	r.State.Push(InModuleBody)
	mod := &Module{name: name, lineNo: lineNo}
	ms := NewMethodSet()
	mod.MethodSet = ms
	ms.Module = mod
	r.MethodSetStack.Push(ms)
	r.moduleStack.Push(mod)
	r.ScopeChain = r.ScopeChain.Extend(mod)
}

func (r *Root) PopModule() *Module {
	module := r.moduleStack.Pop()
	r.MethodSetStack.Pop()
	//classMethodSets[t.Instance.(types.Type)] = class.MethodSet
	r.State.Pop()
	GetType(module, r.ScopeChain, nil)
	r.ScopeChain = r.ScopeChain[:len(r.ScopeChain)-1]
	r.ScopeChain.Set(module.Name(), module)
	module.Parent = r.moduleStack.Peek()
	return module
}

func (r *Root) PushClass(name string, lineNo int) {
	r.State.Push(InClassBody)
	cls := &Class{name: name, lineNo: lineNo, ivars: make(map[string]*IVar)}
	ms := NewMethodSet()
	cls.MethodSet = ms
	ms.Class = cls
	r.currentClass = cls
	r.MethodSetStack.Push(ms)
}

func (r *Root) PopClass() *Class {
	class := r.currentClass
	r.MethodSetStack.Pop()
	r.currentClass = nil
	if parent := r.moduleStack.Peek(); parent != nil {
		parent.Classes = append(parent.Classes, class)
	} else {
		r.Classes = append(r.Classes, class)
	}
	r.inPrivateMethods = false
	t := class.BuildType(r.ScopeChain)
	classMethodSets[t.Instance.(types.Type)] = class.MethodSet
	r.State.Pop()
	r.ScopeChain.Set(class.Name(), class)
	class.Module = r.moduleStack.Peek()
	return class
}

func (r *Root) ParseError() error {
	if len(r.Errors) > 0 {
		return r.Errors[0]
	}
	return nil
}

func (r *Root) AddMethod(m *Method) {
	m.Body.ExplicitReturns = r.ExplicitReturns
	r.ExplicitReturns = []*ReturnNode{}
	ms := r.MethodSetStack.Peek()
	ms.AddMethod(m)
	r.currentMethod = nil
}

func (r *Root) GetMethod(name string) (*Method, bool) {
	if method, ok := r.MethodSetStack.Peek().Methods[name]; ok {
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
	} else if method, found := r.MethodSetStack.Peek().Methods[c.MethodName]; found {
		if err := method.AnalyzeArguments(r.MethodSetStack.Peek().Class, c, nil); err != nil {
			r.AddError(err)
			if e, ok := err.(*ParseError); ok && e.terminal {
				return
			}
		}
	}
	if calls, ok := r.MethodSetStack.Peek().Calls[c.MethodName]; ok {
		r.MethodSetStack.Peek().Calls[c.MethodName] = append(calls, c)
	} else {
		r.MethodSetStack.Peek().Calls[c.MethodName] = []*MethodCall{c}
	}
}

func (r *Root) AddStatement(n Node) {
	switch n.(type) {
	case *Method:
		r.Objects = append(r.Objects, n)
	case *Class, *Module:
		// do nothing, handled differently
	default:
		r.Statements = append(r.Statements, n)
	}
}

func (r *Root) AnalyzeMethodSet(ms *MethodSet, rcvr types.Type) error {
	var err error
	unanalyzedCount := len(ms.Methods)
	for unanalyzedCount > 0 {
		successes := 0
		if initialize, ok := ms.Methods["initialize"]; ok && !initialize.analyzed {
			err = initialize.Analyze(ms)
			if err == nil {
				initialize.analyzed = true
				successes++
			}
		}
		for _, name := range ms.Order {
			m := ms.Methods[name]
			if !m.analyzed {
				err = m.Analyze(ms)
				if err == nil {
					m.analyzed = true
					successes++
				}
			}
		}
		if successes == 0 {
			break
		}
		unanalyzedCount -= successes
	}
	return err
}

func (r *Root) Analyze() error {
	if len(r.Errors) > 0 {
		for _, err := range r.Errors {
			if parseError, ok := err.(*ParseError); ok && parseError.terminal {
				return parseError
			}
		}
	}
	// Okay, current approach is too simplistic. We need to instead
	// Bail on body analysis on error and move onto modules/classes

	// for each module/class incompletely analyzed
	//   analyze method set
	//   while error count > 0 and error count has not changed
	//      analyze all incompletely analyzed methods
	//   if method set is fully analyzed, flag as complete

	// First pass, just to pick up method calls
	if len(r.Statements) > 0 {
		err := (&Body{Statements: r.Statements}).InferReturnType(r.ScopeChain, nil)
		if err != nil {
			if parseError, ok := err.(*ParseError); ok && parseError.terminal {
				return err
			}
		}
	}

	// Work backwards through class declarations so that child classes are
	// analyzed before parents and method calls on inherited methods propagate
	// upward
	for i := len(r.Classes) - 1; i >= 0; i-- {
		class := r.Classes[i]
		if err := r.AnalyzeMethodSet(class.MethodSet, class.Type()); err != nil {
			return err
		}
	}
	if err := r.AnalyzeMethodSet(r.MethodSetStack.Peek(), nil); err != nil {
		return err
	}

	if len(r.Statements) > 0 {
		if err := (&Body{Statements: r.Statements}).InferReturnType(r.ScopeChain, nil); err != nil {
			// probably this is too aggressive
			return err
		}
	}

	for _, calls := range r.MethodSetStack.Peek().Calls {
		for _, c := range calls {
			GetType(c, r.ScopeChain, r.MethodSetStack.Peek().Class)
		}
	}
	return nil
}

func (n *Root) String() string {
	tlos := []Node{}
	if n.Objects != nil {
		tlos = append(tlos, n.Objects...)
	}
	if n.TopLevelModules != nil {
		for _, mod := range n.TopLevelModules {
			tlos = append(tlos, mod)
		}
	}
	if n.Classes != nil {
		for _, cls := range n.Classes {
			tlos = append(tlos, cls)
		}
	}
	if n.Statements != nil {
		tlos = append(tlos, n.Statements...)
	}
	return stdlib.Join[Node](tlos, "\n")
}

type Comment struct {
	Text   string
	LineNo int
}

package parser

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

const (
	Main = "__main__"
)

type RubyLocal struct {
	_type types.Type
	Calls []*MethodCall
}

func (rl *RubyLocal) String() string       { return rl._type.String() }
func (rl *RubyLocal) Type() types.Type     { return rl._type }
func (rl *RubyLocal) SetType(t types.Type) { rl._type = t }
func (rl *RubyLocal) AddCall(c *MethodCall) {
	rl.Calls = append(rl.Calls, c)
}

type Comment struct {
	Text   string
	LineNo int
}

type ParseError struct {
	node Node
	msg  string
}

func (p *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", p.node.LineNo(), p.msg)
}

type MethodSet struct {
	Methods map[string]*Method
	Order   []string
	Calls   map[string][]*MethodCall
	Class   *Class
}

func (ms *MethodSet) AddCall(c *MethodCall) {
	ms.Calls[c.MethodName] = append(ms.Calls[c.MethodName], c)
	cls := ms.Class
	if cls.Parent() != nil {
		cls.Parent().MethodSet.AddCall(c)
	}
}

func NewMethodSet() *MethodSet {
	return &MethodSet{
		Methods: make(map[string]*Method),
		Calls:   make(map[string][]*MethodCall),
	}
}

var globalMethodSet *MethodSet

type Program struct {
	*FSM
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

func NewProgram() *Program {
	globalMethodSet = NewMethodSet()
	p := &Program{
		FSM:        &FSM{},
		MethodSets: []*MethodSet{globalMethodSet},
		Comments:   make(map[int]Comment),
	}
	p.PushScope(NewScope(Main))
	types.ClassRegistry.Initialize()
	return p
}

func (p *Program) AddComment(c Comment) {
	p.Comments[c.LineNo] = c
}

func (p *Program) AddReturn(r *ReturnNode) {
	p.ExplicitReturns = append(p.ExplicitReturns, r)
}

func (p *Program) AddError(err error) {
	p.Errors = append(p.Errors, err)
}

func NewParseError(node Node, fmtString string, args ...interface{}) error {
	return &ParseError{
		node: node,
		msg:  fmt.Sprintf(fmtString, args...),
	}
}

func (p *Program) AddParseError(node Node, fmtString string, args ...interface{}) error {
	err := NewParseError(node, fmtString, args...)
	p.Errors = append(p.Errors, err)
	return err
}

func (p *Program) CurrentString() *StringNode {
	return p.stringStack[len(p.stringStack)-1]
}

func (p *Program) PushString() {
	p.stringStack = append(p.stringStack, &StringNode{Interps: make(map[int][]Node)})
}

func (p *Program) PopString() *StringNode {
	last := p.stringStack[len(p.stringStack)-1]
	p.stringStack = p.stringStack[:len(p.stringStack)-1]
	return last
}

func (p *Program) PushClass(name string, lineNo int) {
	p.PushState(InClassBody)
	cls := &Class{name: name, lineNo: lineNo, ivars: make(map[string]*IVar)}
	ms := NewMethodSet()
	cls.MethodSet = ms
	ms.Class = cls
	p.currentClass = cls
	p.MethodSets = append(p.MethodSets, ms)
}

func (p *Program) PopClass() *Class {
	class := p.currentClass
	p.MethodSets = p.MethodSets[:len(p.MethodSets)-1]
	p.currentClass = nil
	p.Classes = append(p.Classes, class)
	p.inPrivateMethods = false
	t := class.BuildType()
	classMethodSets[t.Instance.(types.Type)] = class.MethodSet
	p.PopState()
	p.ScopeChain.Set(class.Name(), class)
	return class
}

func (p *Program) ParseError() error {
	if len(p.Errors) > 0 {
		return p.Errors[0]
	}
	return nil
}

func (p *Program) CurrentMethodSet() *MethodSet {
	return p.MethodSets[len(p.MethodSets)-1]
}

func (p *Program) AddMethod(m *Method) {
	m.Body.ExplicitReturns = p.ExplicitReturns
	p.ExplicitReturns = []*ReturnNode{}
	ms := p.CurrentMethodSet()
	ms.Methods[m.Name] = m
	ms.Order = append(ms.Order, m.Name)
	p.currentMethod = nil
}

func (p *Program) GetMethod(name string) (*Method, bool) {
	if method, ok := p.CurrentMethodSet().Methods[name]; ok {
		return method, true
	} else {
		return nil, false
	}
}

func (p *Program) AddCall(c *MethodCall) {
	if c.Receiver != nil {
		switch r := c.Receiver.(type) {
		case *IdentNode:
			loc := p.ScopeChain.ResolveVar(r.Val)
			if loc != BadLocal {
				loc.(*RubyLocal).AddCall(c)
			} else {
				uncalled := &RubyLocal{}
				uncalled.AddCall(c)
				p.ScopeChain.Set(r.Val, uncalled)
				return
			}
		}
	} else if method, found := p.CurrentMethodSet().Methods[c.MethodName]; found {
		if err := method.AnalyzeArguments(p.CurrentMethodSet().Class, c); err != nil {
			p.AddError(err)
			return
		}
	}
	if calls, ok := p.CurrentMethodSet().Calls[c.MethodName]; ok {
		p.CurrentMethodSet().Calls[c.MethodName] = append(calls, c)
	} else {
		p.CurrentMethodSet().Calls[c.MethodName] = []*MethodCall{c}
	}
}

func (p *Program) AddStatement(n Node) {
	switch n.(type) {
	case *Method:
		p.Objects = append(p.Objects, n)
	case *Class:
		// do nothing, handled differently
	default:
		p.Statements = append(p.Statements, n)
	}
}

func (p *Program) AnalyzeMethodSet(ms *MethodSet, rcvr types.Type) error {
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

func (p *Program) Analyze() error {
	if len(p.Errors) > 0 {
		return p.Errors[0]
	}
	// First pass, just to pick up method calls
	if len(p.Statements) > 0 {
		if err := (&Body{Statements: p.Statements}).InferReturnType(p.ScopeChain, nil); err != nil {
			// probably this is too aggressive
			return err
		}
	}

	for i := len(p.Classes) - 1; i >= 0; i-- {
		class := p.Classes[i]
		if err := p.AnalyzeMethodSet(class.MethodSet, class.Type()); err != nil {
			return err
		}
	}
	if err := p.AnalyzeMethodSet(p.CurrentMethodSet(), nil); err != nil {
		return err
	}

	for _, calls := range p.CurrentMethodSet().Calls {
		for _, c := range calls {
			GetType(c, p.ScopeChain, p.CurrentMethodSet().Class)
		}
	}
	return nil
}

func (n *Program) String() string {
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

type Node interface {
	String() string
	TargetType(ScopeChain, *Class) (types.Type, error)
	Type() types.Type
	SetType(types.Type)
	LineNo() int
}

func GetType(n Node, scope ScopeChain, class *Class) (t types.Type, err error) {
	t = n.Type()
	if t == nil {
		if ident, ok := n.(*IdentNode); ok {
			if loc := scope.ResolveVar(ident.Val); loc != BadLocal {
				ident.SetType(loc.Type())
			} else if m, ok := globalMethodSet.Methods[ident.Val]; ok {
				if err := m.Analyze(globalMethodSet); err != nil {
					return nil, err
				}
				ident.MethodCall = &MethodCall{
					Method:     m,
					MethodName: m.Name,
					_type:      m.ReturnType(),
					lineNo:     ident.lineNo,
				}
				return m.ReturnType(), nil
			}
		}
		if t, err = n.TargetType(scope, class); err != nil {
			return nil, err
		} else {
			n.SetType(t)
			return t, nil
		}
	}
	return t, nil
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

type InfixExpressionNode struct {
	Operator string
	Left     Node
	Right    Node
	lineNo   int
	_type    types.Type
}

func (n *InfixExpressionNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left, n.Operator, n.Right)
}
func (n *InfixExpressionNode) Type() types.Type     { return n._type }
func (n *InfixExpressionNode) SetType(t types.Type) { n._type = t }
func (n *InfixExpressionNode) LineNo() int          { return n.lineNo }

func (n *InfixExpressionNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	tl, err := GetType(n.Left, locals, class)
	if err != nil {
		return nil, err
	}
	tr, err := GetType(n.Right, locals, class)
	if err != nil {
		return nil, err
	}
	if n.HasMethod() {
		if t, err := tl.MethodReturnType(n.Operator, nil, []types.Type{tr}); err != nil {
			return nil, NewParseError(n, err.Error())
		} else {
			return t, nil
		}
	}
	return nil, NewParseError(n, "No method `%s` on type %s", n.Operator, tl)
}

func (n *InfixExpressionNode) HasMethod() bool {
	if n.Left.Type() != nil {
		return n.Left.Type().HasMethod(n.Operator)
	}
	return false
}

type NotExpressionNode struct {
	Arg    Node
	lineNo int
	_type  types.Type
}

func (n *NotExpressionNode) String() string       { return fmt.Sprintf("!%s", n.Arg) }
func (n *NotExpressionNode) Type() types.Type     { return n._type }
func (n *NotExpressionNode) SetType(t types.Type) { n._type = types.BoolType }
func (n *NotExpressionNode) LineNo() int          { return n.lineNo }

func (n *NotExpressionNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if _, err := GetType(n.Arg, locals, class); err != nil {
		return nil, err
	}
	return types.BoolType, nil
}

type AssignmentNode struct {
	Left         []Node
	Right        []Node
	Reassignment bool
	OpAssignment bool
	lineNo       int
	_type        types.Type
}

func (n *AssignmentNode) String() string {
	sides := []interface{}{}
	for _, side := range [][]Node{n.Left, n.Right} {
		segments := []string{}
		for _, s := range side {
			segments = append(segments, s.String())
		}
		var s string
		if len(side) > 1 {
			s = fmt.Sprintf("(%s)", strings.Join(segments, ", "))
		} else {
			s = side[0].String()
		}
		sides = append(sides, s)
	}
	return fmt.Sprintf("(%s = %s)", sides...)
}
func (n *AssignmentNode) Type() types.Type     { return n._type }
func (n *AssignmentNode) SetType(t types.Type) { n._type = t }
func (n *AssignmentNode) LineNo() int          { return n.lineNo }

func (n *AssignmentNode) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	var typelist []types.Type
	for i, left := range n.Left {
		var localName string
		switch lhs := left.(type) {
		case *IdentNode:
			localName = lhs.Val
			GetType(lhs, scope, class)
		case *BracketAssignmentNode:
			localName = lhs.Composite.(*IdentNode).Val
		case *IVarNode:
			GetType(lhs, scope, class)
		case *ConstantNode:
			localName = lhs.Val
		default:
			return nil, NewParseError(lhs, "%s not yet supported in LHS of assignments", lhs)
		}
		var (
			assignedType types.Type
			err          error
		)
		if n.OpAssignment {
			// operator assignments are always 1:1, so nothing to handle here for multiple lhs or rhs
			assignedType, err = GetType(n.Right[i].(*InfixExpressionNode).Right, scope, class)
		} else {
			switch {
			case len(n.Left) > len(n.Right):
				/*

					There are two valid scenarios here: unpacking of an array into
					locals, and assigning from a method that returns a tuple. Note that
					Ruby's behavior in the event of a length mismatch of the two sides is
					to drop the excess values if lhs is shorter than rhs, and to populate
					excess identifiers on lhs with nil of lhs is longer than rhs. There
					is also the perfectly legal option of assigning a single value that
					cannot be deconstructed to multiple variables, which leaves all but
					the first as nil.

					This logic will have to change to accommodate splats.

				*/
				t, err := GetType(n.Right[0], scope, class)
				if err != nil {
					return nil, NewParseError(n, err.Error())
				}
				switch rt := t.(type) {
				case types.Multiple:
					assignedType = rt[i]
				case types.Array:
					assignedType = rt.Element
				default:
					if i > 0 {
						assignedType = types.NilType
					} else {
						assignedType = t
					}
				}
			case len(n.Left) == len(n.Right):
				assignedType, err = GetType(n.Right[i], scope, class)
			case len(n.Left) < len(n.Right):
				// If there's only one lhs element, this is an implicit Array, and
				// needs to get type checked. Otherwise, as discussed above, we throw
				// away any rhs values beyond the length of lhs.
				if len(n.Left) == 1 {
					array := &ArrayNode{Args: ArgsNode(n.Right), lineNo: n.Right[0].LineNo()}
					if at, err := GetType(array, scope, class); err != nil {
						return nil, err
					} else {
						n.Right = []Node{array}
						assignedType = at
					}
				} else {
					assignedType, err = GetType(n.Right[i], scope, class)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		switch lft := left.(type) {
		case *IVarNode:
			lft.SetType(assignedType)
			n.Reassignment = true
		case *ConstantNode:
			lft.SetType(assignedType)
			if class != nil {
				constant := &Constant{Name: lft.Val}
				constant._type = assignedType
				GetType(left, scope, class)
				constant.Val = n.Right[i]
				constant.Class = class
				class.Constants = append(class.Constants, constant)
			} else {
				scope.Set(localName, &RubyLocal{_type: assignedType})
			}
		default:
			if local, found := scope.Get(localName); !found {
				scope.Set(localName, &RubyLocal{_type: assignedType})
			} else {
				if local.Type() == nil {
					loc := local.(*RubyLocal)
					loc.SetType(assignedType)
					for _, c := range loc.Calls {
						GetType(c, scope, class)
					}
				} else {
					n.Reassignment = true
				}
				if local.Type() != assignedType {
					if arr, ok := local.Type().(types.Array); ok {
						if arr.Element != assignedType {
							return nil, NewParseError(n, "Attempted to assign %s member to %s", assignedType, arr)
						}
					} else {
						return nil, NewParseError(n, "tried assigning type %s to local %s in scope %s but had previously assigned type %s", assignedType, localName, scope.Name(), scope.MustGet(localName))
					}
				}
			}
		}
		typelist = append(typelist, assignedType)
	}
	if len(typelist) > 1 {
		return types.Multiple(typelist), nil
	}
	return typelist[0], nil
}

func (n *AssignmentNode) Clone() *AssignmentNode {
	return &AssignmentNode{
		Left:         n.Left,
		Right:        n.Right,
		Reassignment: n.Reassignment,
		OpAssignment: n.OpAssignment,
		lineNo:       n.lineNo,
		_type:        n._type,
	}
}

type MethodCall struct {
	Receiver   Node
	Method     *Method
	MethodName string
	Args       ArgsNode
	Block      *Block
	RawBlock   string
	Getter     bool
	_type      types.Type
	lineNo     int
}

func (n *MethodCall) String() string {
	var s string
	args := []string{}
	if len(n.Args) > 0 {
		args = append(args, n.Args.String())
	}
	if n.Block != nil {
		args = append(args, "block = "+n.Block.String())
	}
	s = fmt.Sprintf("%s(%s)", n.MethodName, strings.Join(args, ", "))

	if n.Receiver != nil {
		s = n.Receiver.String() + "." + s
	}
	return fmt.Sprintf("(%s)", s)
}

func (n *MethodCall) Type() types.Type     { return n._type }
func (n *MethodCall) SetType(t types.Type) { n._type = t }
func (n *MethodCall) LineNo() int          { return n.lineNo }

func (c *MethodCall) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	receiverType := c.ReceiverType(scope, class)
	switch t := receiverType.(type) {
	case *types.Class:
		if c.MethodName == "new" && t.UserDefined {
			receiverType := t.Instance.(types.Type)
			initializeCall := &MethodCall{
				MethodName: "initialize",
				Args:       c.Args,
				Block:      c.Block,
				_type:      receiverType,
				lineNo:     c.lineNo,
			}
			classMethodSets[receiverType].AddCall(initializeCall)
		}
	case types.Instance:
		// We'll only have a methodset for a user-defined class instance type
		if ms, ok := classMethodSets[t]; ok {
			ms.AddCall(c)
		}
	case *types.Proc:
		if c.MethodName == "call" {
			localName := c.Receiver.(*IdentNode).Val
			if local := scope.ResolveVar(localName); local != BadLocal {
				blk := local.(*Block)
				for i, arg := range c.Args {
					if t, err := GetType(arg, scope, class); err != nil {
						return nil, err
					} else {
						p, err := blk.GetParam(i)
						if err != nil {
							return nil, NewParseError(c, err.Error())
						}
						p._type = t
						method := blk.Method
						method.Block.AddParam(p)
						blk.Scope.Set(p.Name, &RubyLocal{_type: t})
					}
				}
				err := blk.Body.InferReturnType(blk.Scope, nil)
				if err != nil {
					return nil, err
				}
				blk.Method.Block.ReturnType = blk.Body.ReturnType
				return blk.Body.ReturnType, nil
			}
		}
	}
	if c.Receiver != nil {
		if receiverType == nil {
			return nil, fmt.Errorf("Method '%s' called on '%s' but type of '%s' is not inferred", c.MethodName, c.Receiver, c.Receiver)
		}
		if !receiverType.HasMethod(c.MethodName) {
			if ms, ok := classMethodSets[receiverType]; ok && ms.Class != nil && len(c.Args) == 0 {
				for _, ivar := range ms.Class.IVars(nil) {
					if c.MethodName == ivar.Name && ivar.Readable {
						c.Getter = true
						return ivar.Type(), nil
					}
				}
			}
			return nil, NewParseError(c, "No known method '%s' on %s", c.MethodName, receiverType)
		}
	}
	argTypes := []types.Type{}
	for _, a := range c.Args {
		if t, err := GetType(a, scope, class); err != nil {
			return nil, err
		} else {
			argTypes = append(argTypes, t)
		}
	}

	var method *Method

	if ms, ok := classMethodSets[receiverType]; ok {
		if m, userDefined := ms.Methods[c.MethodName]; userDefined {
			method = m
		}
	} else if c.Receiver == nil {
		if class == nil {
			method = globalMethodSet.Methods[c.MethodName]
		} else {
			//TODO push into class methods when class method resolution is implemented
			switch c.MethodName {
			case "attr_reader":
				class.AddIVars(c.Args, true, false)
				delete(class.MethodSet.Calls, c.MethodName)
			case "attr_writer":
				class.AddIVars(c.Args, false, true)
				delete(class.MethodSet.Calls, c.MethodName)
			case "attr_accessor":
				class.AddIVars(c.Args, true, true)
				delete(class.MethodSet.Calls, c.MethodName)
			default:
				return nil, NewParseError(c, "Tried calling class method '%s' inside body of class '%s' but no such method exists", c.MethodName, class.Name())
			}
			return nil, nil
		}
	}

	var blockRetType types.Type
	/*
		TODO if a block is given, which we should be able to determine right now, we
		can't plow straight through `InferReturnType`. Instead, we need to:

		* run InferReturnType down `blk.call` so that we can determine the types of the arguments to the block
		* set those types on the block (which means having it available)
		* using the types obtained for the block args, get the return type for the block
		* resume inference where we left off at bullet #1
	*/
	if method != nil {
		//TODO should be consolidated with AnalyzeArguments/AnalyzeMethodSet
		c.Method = method
		for i, t := range argTypes {
			param, _ := method.GetParam(i)
			param._type = t
			method.Locals.Set(param.Name, &RubyLocal{_type: param.Type()})
		}
		if c.Block != nil {
			c.Block.Scope = scope.Extend(NewScope("block"))
			c.Block.Method = method
			method.Locals.Set(method.Block.Name, c.Block)
		}
		// set block in scope here
		if err := method.Body.InferReturnType(method.Scope, nil); err != nil {
			return nil, err
		} else {
			return method.ReturnType(), nil
		}
	} else if c.Receiver == nil {
		return nil, NewParseError(c, "Attempted to call undefined method '%s'", c.MethodName)
	} else {
		// This is all a special case for thanos-defined methods
		if c.Block != nil {
			blockScope := NewScope("block")
			blockArgTypes := receiverType.BlockArgTypes(c.MethodName, argTypes)
			for i, p := range c.Block.Params {
				blockScope.Set(p.Name, &RubyLocal{_type: blockArgTypes[i]})
			}
			err := c.Block.Body.InferReturnType(scope.Extend(blockScope), nil)
			if err != nil {
				return nil, err
			}
			blockRetType = c.Block.Body.ReturnType
		}
	}

	if t, err := receiverType.MethodReturnType(c.MethodName, blockRetType, argTypes); err != nil {
		return nil, NewParseError(c, err.Error())
	} else {
		return t, nil
	}
}

func (n *MethodCall) RequiresTransform() bool {
	if n.Receiver == nil {
		return false // for now, will have some built-in top level funcs
	}

	return n.Receiver.Type().HasMethod(n.MethodName)
}

func (c *MethodCall) ReceiverType(scope ScopeChain, class *Class) types.Type {
	if c.Receiver != nil {
		if c.Receiver.Type() != nil {
			return c.Receiver.Type()
		}
		receiverType, err := GetType(c.Receiver, scope, class)
		if err == nil {
			return receiverType
		}
	}
	if types.KernelType.HasMethod(c.MethodName) {
		c.Receiver = &KernelNode{}
		return types.KernelType
	}
	return nil
}

func (c *MethodCall) PositionalArgs() ArgsNode {
	positional := ArgsNode{}
	for _, a := range c.Args {
		if _, ok := a.(*KeyValuePair); !ok {
			positional = append(positional, a)
		}
	}
	return positional
}

func (c *MethodCall) SetBlock(blk *Block) {
	c.Block = blk
	if c.Method != nil {
		for _, p := range blk.Params {
			c.Method.Block.AddParam(p)
		}
	}
}

type ArgsNode []Node

func (n ArgsNode) String() string {
	strs := []string{}
	for _, s := range n {
		strs = append(strs, s.String())
	}
	return strings.Join(strs, ", ")
}

func (n ArgsNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	panic("ArgsNode#TargetType should never be called")
	return n[0].TargetType(locals, class)
}

// Wrong but dummy for satisfying interface
func (n ArgsNode) Type() types.Type     { return n[0].Type() }
func (n ArgsNode) SetType(t types.Type) {}
func (n ArgsNode) LineNo() int          { return 0 }

func (n ArgsNode) FindByName(name string) (Node, error) {
	for _, arg := range n {
		if kv, ok := arg.(*KeyValuePair); ok && kv.Label == name {
			return kv, nil
		}
	}
	return nil, fmt.Errorf("No argument named '%s' found", name)
}

type ParamList struct {
	Params   []*Param
	ParamMap map[string]*Param
}

func NewParamList() *ParamList {
	return &ParamList{ParamMap: make(map[string]*Param)}
}

func (list *ParamList) AddParam(p *Param) error {
	if _, found := list.ParamMap[p.Name]; found {
		return fmt.Errorf("parameter '%s' declared twice", p.Name)
	}
	list.Params = append(list.Params, p)
	list.ParamMap[p.Name] = p
	p.Position = len(list.Params) - 1
	return nil
}

func (list *ParamList) GetParam(i int) (*Param, error) {
	if i < len(list.Params) {
		return list.Params[i], nil
	}
	return nil, errors.New("out of bounds")
}

func (list *ParamList) PositionalParams() []*Param {
	params := []*Param{}
	for i := 0; ; i++ {
		p, err := list.GetParam(i)
		if err != nil || p.Kind != Positional {
			break
		}
		params = append(params, p)
	}
	return params
}

func (list *ParamList) GetParamByName(s string) *Param {
	return list.ParamMap[s]
}

type BlockParam struct {
	Name       string
	ReturnType types.Type
	*ParamList
}

type Method struct {
	Receiver Node
	Name     string
	Body     *Body
	*ParamList
	Locals  *SimpleScope
	Scope   ScopeChain
	Program *Program
	Block   *BlockParam
	lineNo  int
	Private bool
}

func NewMethod(name string, p *Program) *Method {
	locals := NewScope(name)
	p.currentMethod = &Method{
		Name:      name,
		ParamList: NewParamList(),
		Locals:    locals,
		Scope:     p.ScopeChain.Extend(locals),
		Program:   p,
	}
	return p.currentMethod
}

func (n *Method) String() string {
	strs := []string{}
	for _, p := range n.Params {
		strs = append(strs, p.Name)
	}
	if n.Block != nil {
		strs = append(strs, "&"+n.Block.Name)
	}

	if n.Receiver != nil {
		return fmt.Sprintf("(def %s#%s(%s); %s; end)", n.Receiver, n.Name, strings.Join(strs, ", "), n.Body)
	} else {
		return fmt.Sprintf("(def %s(%s); %s; end)", n.Name, strings.Join(strs, ", "), n.Body)
	}
}

func (n *Method) Type() types.Type     { return types.FuncType }
func (n *Method) SetType(t types.Type) {}
func (n *Method) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.FuncType, nil
}
func (n *Method) LineNo() int { return n.lineNo }

func (m *Method) ReturnType() types.Type {
	return m.Body.ReturnType
}

func (m *Method) GoName() string {
	name := strings.TrimRight(m.Name, "?!")
	if !m.Private {
		name = strings.Title(name)
	}
	return name
}

func (m *Method) AddParam(p *Param) error {
	if p.Kind == ExplicitBlock {
		m.Block = &BlockParam{Name: p.Name, ParamList: NewParamList()}
		return nil
	}
	err := m.ParamList.AddParam(p)
	if err != nil {
		return NewParseError(m, err.Error())
	}
	m.Locals.Set(p.Name, &RubyLocal{})
	return nil
}

func (m *Method) Analyze(ms *MethodSet) error {
	for _, c := range ms.Calls[m.Name] {
		if err := m.AnalyzeArguments(ms.Class, c); err != nil {
			return err
		}
	}
	for _, param := range m.Params {
		if param.Type() == nil {
			name := m.Name
			if ms.Class != nil {
				name = ms.Class.Name() + "#" + name
			}
			return NewParseError(m, "unable to detect type signature of method '%s' because it is never called", name)
		}
		m.Locals.Set(param.Name, &RubyLocal{_type: param.Type()})
	}
	if err := m.Body.InferReturnType(m.Scope, ms.Class); err != nil {
		return err
	}
	for _, c := range ms.Calls[m.Name] {
		c.Method = m
		if c.Type() == nil {
			c.SetType(m.ReturnType())
		}
	}
	return nil
}

func (method *Method) AnalyzeArguments(class *Class, c *MethodCall) error {
	for _, p := range method.Params {
		if p.Default != nil {
			t, err := GetType(p.Default, ScopeChain{class}, class)
			if err != nil {
				return err
			}
			//TODO this is happening in at least three places
			method.Locals.Set(p.Name, &RubyLocal{_type: t})
		}
	}
	if c == nil {
		return nil
	}
	if len(method.PositionalParams()) > len(c.PositionalArgs()) {
		return NewParseError(c, "method '%s' called with %d positional arguments but %d expected", method.Name, len(c.PositionalArgs()), len(method.PositionalParams()))
	}
	for i, arg := range c.Args {
		var param *Param
		if kv, ok := arg.(*KeyValuePair); ok {
			param = method.GetParamByName(kv.Label)
			if param == nil {
				return NewParseError(c, "method '%s' called with keyword argument '%s' but '%s' has no such parameter", method.Name, kv.Label, method.Name)
			}
		} else {
			var err error
			param, err = method.GetParam(i)
			if err != nil {
				return NewParseError(c, "method '%s' called with %d arguments but %d expected", method.Name, i+1, i)
			}
		}
		if param.Type() == nil {
			// unset, so set it
			if t, err := GetType(arg, method.Scope, class); err != nil {
				return err
			} else {
				param._type = t
			}
		} else {
			t, err := GetType(arg, method.Scope, class)
			if err == nil && t != param.Type() {
				return NewParseError(c, "method '%s' called with %s for parameter '%s' but '%s' was previously seen as %s", method.Name, t, param.Name, param.Name, param.Type())
			}
		}
	}
	return nil
}

type ReturnNode struct {
	Val    ArgsNode
	_type  types.Type
	lineNo int
}

func (n *ReturnNode) String() string       { return fmt.Sprintf("(return %s)", n.Val) }
func (n *ReturnNode) Type() types.Type     { return n._type }
func (n *ReturnNode) SetType(t types.Type) { n._type = t }
func (n *ReturnNode) LineNo() int          { return n.lineNo }

func (n *ReturnNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if len(n.Val) == 1 {
		return GetType(n.Val[0], locals, class)
	}
	multiple := types.Multiple{}
	for _, single := range n.Val {
		t, err := GetType(single, locals, class)
		if err != nil {
			return t, err
		}
		multiple = append(multiple, t)
	}
	return multiple, nil
}

type Statements []Node

func (stmts Statements) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	var lastReturnedType types.Type
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *AssignmentNode:
			if t, err := GetType(s, scope, class); err != nil {
				return nil, err
			} else {
				lastReturnedType = t
			}
		case *Condition:
			// We need this to be semi-"live" since otherwise we can't surface an
			// error about a type mismatch between the branches. The type on the
			// condition will still be effectively memoized since it can just get the
			// cached value from the True side. Thus we call TargetType directly on
			// the node instead of going through GetType.
			if t, err := GetType(s, scope, class); err != nil {
				return nil, err
			} else {
				lastReturnedType = t
			}
		case *IVarNode:
			if t, err := GetType(s, scope, class); err != nil {
				return nil, err
			} else {
				lastReturnedType = t
			}
		default:
			if c, ok := stmt.(*MethodCall); ok {
				// Handle method chaining -- walk down to the first identifier or
				// literal and infer types on the way back up so that receiver type is
				// known for each subsequent method call
				chain := []Node{c}
				r := c.Receiver
				walking := true
				for walking {
					switch c := r.(type) {
					case *MethodCall:
						chain = append(chain, c)
						r = c.Receiver
					default:
						if c != nil {
							chain = append(chain, c)
						}
						walking = false
					}
				}
				for i := len(chain) - 1; i >= 0; i-- {
					if t, err := GetType(chain[i], scope, class); err != nil {
						return nil, err
					} else {
						lastReturnedType = t
					}
				}
			} else if t, err := GetType(stmt, scope, class); err != nil {
				return nil, err
			} else {
				lastReturnedType = t
			}
		}
	}
	return lastReturnedType, nil
}

func (stmts Statements) String() string {
	strs := []string{}
	for _, s := range stmts {
		strs = append(strs, s.String())
	}
	switch len(strs) {
	case 0:
		return ""
	case 1:
		return strs[0]
	default:
		return fmt.Sprintf("(%s)", strings.Join(strs, "; "))
	}
}

func (stmts Statements) Type() types.Type     { return nil }
func (stmts Statements) SetType(t types.Type) {}
func (stmts Statements) LineNo() int          { return 0 }

type Body struct {
	Statements      Statements
	ReturnType      types.Type
	ExplicitReturns []*ReturnNode
}

func (b *Body) InferReturnType(scope ScopeChain, class *Class) error {
	// To guess the right return type of a method, we have to:

	//	1) track all return statements in the method body;

	//  2) chase expressions all the way to the end of the body and wrap that
	//  last expr in a return node if it's not already there, wherein we record
	//  the types of all assignments in a map on the method.

	// Achieving 1) would mean rewalking this branch of the AST right after
	// building it which seems dumb, so instead we register each ReturnNode on
	// the method as the parser encounters them so we can loop through them
	// afterward when m.Locals is fully populated.

	lastReturnedType, err := GetType(b.Statements, scope, class)
	if err != nil {
		return err
	}
	finalStatementIdx := len(b.Statements) - 1
	finalStatement := b.Statements[finalStatementIdx]
	switch s := finalStatement.(type) {
	case *ReturnNode:
	case *AssignmentNode:
		var ret *ReturnNode
		if s.OpAssignment {
			ret = &ReturnNode{Val: s.Left}
		} else if _, ok := s.Left[0].(*IVarNode); ok {
			ret = &ReturnNode{Val: s.Left}
		} else {
			ret = &ReturnNode{Val: []Node{s.Right[0]}}
		}
		if _, err := GetType(ret, scope, class); err != nil {
			return err
		}
		b.Statements = append(b.Statements, ret)
	default:
		if finalStatement.Type() != types.NilType && scope.Name() != Main {
			ret := &ReturnNode{Val: []Node{finalStatement}}
			if _, err := GetType(ret, scope, class); err != nil {
				return err
			}
			b.Statements[finalStatementIdx] = ret
		}
	}
	if len(b.ExplicitReturns) > 0 {
		for _, r := range b.ExplicitReturns {
			t, _ := GetType(r, scope, class)
			if !t.Equals(lastReturnedType) {
				return NewParseError(r, "Detected conflicting return types %s and %s in method '%s'", lastReturnedType, t, scope.Name())
			}
		}
	}
	b.ReturnType = lastReturnedType
	return nil
}

func (n *Body) String() string {
	return n.Statements.String()
}

type ParamKind int

const (
	Positional ParamKind = iota
	Named
	Keyword
	ExplicitBlock
)

type Param struct {
	Position int
	Name     string
	Kind     ParamKind
	_type    types.Type
	Default  Node
	Required bool
}

func (p *Param) Type() types.Type {
	if p.Default != nil {
		return p.Default.Type()
	}
	return p._type
}

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

type Block struct {
	Body   *Body
	Scope  ScopeChain
	Method *Method
	*ParamList
}

func (b *Block) String() string {
	strs := []string{}
	for _, p := range b.Params {
		strs = append(strs, p.Name)
	}

	return fmt.Sprintf("(|%s| %s)", strings.Join(strs, ", "), b.Body)
}

func (b *Block) Type() types.Type {
	return types.NewProc()
}

type StringKind int

const (
	DoubleQuote StringKind = iota
	SingleQuote
	Regexp
)

var stringDelims = map[StringKind]string{
	DoubleQuote: `"`,
	SingleQuote: "'",
	Regexp:      "/",
}

type StringNode struct {
	BodySegments []string
	Interps      map[int][]Node
	cached       bool
	Kind         StringKind
	lineNo       int
}

func (n *StringNode) OrderedInterps() []Node {
	positions := []int{}
	for k, _ := range n.Interps {
		positions = append(positions, k)
	}
	sort.Ints(positions)
	nodes := []Node{}
	for _, i := range positions {
		interp := n.Interps[i]
		nodes = append(nodes, interp...)
	}
	return nodes
}

func (n *StringNode) GoString() string {
	switch n.Kind {
	case Regexp:
		return strings.ReplaceAll(n.FmtString("`"), "(?<", "(?P<")
	case SingleQuote:
		return n.FmtString("`")
	default:
		return n.FmtString(`"`)
	}
}

func (n *StringNode) FmtString(delim string) string {
	if len(n.Interps) == 0 {
		if len(n.BodySegments) == 0 {
			return delim + delim
		}
		return delim + n.BodySegments[0] + delim
	}
	segments := ""
	for i, seg := range n.BodySegments {
		if interps, exists := n.Interps[i]; exists {
			for _, interp := range interps {
				verb := types.FprintVerb(interp.Type())
				if verb == "" {
					panic(fmt.Sprintf("[line %d] Unhandled type inference failure for interpolated value in string", n.lineNo))
				}
				segments += verb
			}
		}
		segments += seg
	}
	if trailingInterps, exists := n.Interps[len(n.BodySegments)]; exists {
		for _, trailingInterp := range trailingInterps {
			verb := types.FprintVerb(trailingInterp.Type())
			if verb == "" {
				panic(fmt.Sprintf("[line %d] Unhandled type inference failure for interpolated value in string", n.lineNo))
			}
			segments += verb
		}
	}
	return delim + segments + delim
}

func (n *StringNode) String() string {
	interps := []string{}
	for _, interp := range n.OrderedInterps() {
		interps = append(interps, interp.String())
	}
	if len(n.Interps) == 0 {
		return n.FmtString(stringDelims[n.Kind])
	}
	return fmt.Sprintf(`(%s %% (%s))`, n.FmtString(stringDelims[n.Kind]), strings.Join(interps, ", "))
}

func (n *StringNode) Type() types.Type {
	if len(n.Interps) == 0 || n.cached {
		switch n.Kind {
		case Regexp:
			return types.RegexpType
		default:
			return types.StringType
		}
	}
	return nil
}

func (n *StringNode) SetType(t types.Type) {}
func (n *StringNode) LineNo() int          { return n.lineNo }

func (n *StringNode) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	for _, interps := range n.Interps {
		for _, i := range interps {
			if t, err := GetType(i, scope, class); err != nil {
				if t == nil {
					return nil, NewParseError(n, "Could not infer type for interpolated value '%s'", i)
				}
				return nil, err
			}
		}
	}
	n.cached = true
	return types.StringType, nil
}

// Placeholder in AST for Kernel method lookups
type KernelNode struct{}

func (n *KernelNode) String() string       { return "Kernel" }
func (n *KernelNode) Type() types.Type     { return types.KernelType }
func (n *KernelNode) SetType(t types.Type) {}
func (n *KernelNode) LineNo() int          { return 0 }

func (n *KernelNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.KernelType, nil
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

type UnimplementedNode struct {
	lineNo int
}

func (n *UnimplementedNode) String() string       { return "nil" }
func (n *UnimplementedNode) Type() types.Type     { return types.NilType }
func (n *UnimplementedNode) SetType(t types.Type) {}
func (n *UnimplementedNode) LineNo() int          { return n.lineNo }

func (n *UnimplementedNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
}

type RangeNode struct {
	Lower, Upper Node
	Inclusive    bool
	lineNo       int
	_type        types.Type
}

func (n *RangeNode) String() string {
	rangeOp := "..."
	if n.Inclusive {
		rangeOp = ".."
	}
	upper := ""
	if n.Upper != nil {
		upper = n.Upper.String()
	}
	return fmt.Sprintf("(%s%s%s)", n.Lower, rangeOp, upper)
}
func (n *RangeNode) Type() types.Type     { return n._type }
func (n *RangeNode) SetType(t types.Type) { n._type = types.RangeType }
func (n *RangeNode) LineNo() int          { return n.lineNo }

func (n *RangeNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	for _, bound := range []Node{n.Lower, n.Upper} {
		if bound != nil {
			t, err := GetType(bound, locals, class)
			if err != nil {
				return t, err
			}
			if t != types.IntType {
				return t, NewParseError(n, "Tried to construct range with %s but only IntType is allowed", t)
			}
		}
	}
	return types.RangeType, nil
}

type CaseNode struct {
	Value             Node
	Whens             []*WhenNode
	RequiresExpansion bool
	_type             types.Type
	lineNo            int
}

func (n *CaseNode) String() string {
	segments := []string{}
	for _, when := range n.Whens {
		segments = append(segments, when.String())
	}
	return fmt.Sprintf("(case %s %s)", n.Value, strings.Join(segments, "; "))
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

type NoopNode struct {
	lineNo int
}

func (n *NoopNode) String() string       { return "nil" }
func (n *NoopNode) Type() types.Type     { return types.NilType }
func (n *NoopNode) SetType(t types.Type) {}
func (n *NoopNode) LineNo() int          { return n.lineNo }

func (n *NoopNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
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

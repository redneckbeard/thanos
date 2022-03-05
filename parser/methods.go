package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

var setterPatt = regexp.MustCompile(`\w+=`)

type MethodSet struct {
	Methods map[string]*Method
	Order   []string
	Calls   map[string][]*MethodCall
	Class   *Class
}

func (ms *MethodSet) AddMethod(m *Method) {
	ms.Methods[m.Name] = m
	ms.Order = append(ms.Order, m.Name)
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
	Root    *Root
	Block   *BlockParam
	lineNo  int
	Private bool
}

func NewMethod(name string, r *Root) *Method {
	locals := NewScope(name)
	r.currentMethod = &Method{
		Name:      name,
		ParamList: NewParamList(),
		Locals:    locals,
		Scope:     r.ScopeChain.Extend(locals),
		Root:      r,
	}
	return r.currentMethod
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
	if setterPatt.MatchString(name) {
		name = "Set" + strings.TrimRight(name, "=")
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
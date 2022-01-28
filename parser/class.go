package parser

import (
	"fmt"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

var classMethodSets = make(map[types.Type]*MethodSet)

type IVarNode struct {
	Val    string
	Class  *Class
	_type  types.Type
	lineNo int
}

func (n *IVarNode) String() string   { return n.Val }
func (n *IVarNode) Type() types.Type { return n._type }
func (n *IVarNode) SetType(t types.Type) {
	n._type = t
	if n.Class != nil {
		if ivar := n.IVar(); ivar != nil {
			ivar._type = t
		}
	} else {
		panic("Setting an instance variable outside of a class is unsupported")
	}
}
func (n *IVarNode) LineNo() int { return n.lineNo }

func (n *IVarNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if class != nil {
		n.Class = class
		if spec, exists := n.Class.ivars[n.NormalizedVal()]; exists {
			return spec.Type(), nil
		}
	}
	return nil, NewParseError(n, "Unable to connect to the mothership")
}

func (n *IVarNode) Copy() Node {
	// just a guess but these probably don't need to be copied
	return n
}

func (n *IVarNode) NormalizedVal() string {
	//TODO lexer/parser bug here where in isolation ivar tokens come back as `@foo\n`
	return strings.TrimSpace(strings.TrimLeft(n.Val, "@"))
}

func (n *IVarNode) IVar() *IVar {
	if ivar, exists := n.Class.ivars[n.NormalizedVal()]; exists {
		return ivar
	}
	return nil
}

type CVarNode struct {
	Val    string
	_type  types.Type
	lineNo int
}

func (n *CVarNode) String() string       { return n.Val }
func (n *CVarNode) Type() types.Type     { return n._type }
func (n *CVarNode) SetType(t types.Type) { n._type = t }
func (n *CVarNode) LineNo() int          { return n.lineNo }

func (n *CVarNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return nil, nil
}

func (n *CVarNode) Copy() Node {
	return n
}

type IVar struct {
	Name                string
	_type               types.Type
	Readable, Writeable bool
}

func (ivar *IVar) Type() types.Type {
	return ivar._type
}

type Const interface {
	Constant()
	Name() string
	Type() types.Type
}

type Constant struct {
	name      string
	prefix    string
	Val       Node
	Namespace Namespace
	_type     types.Type
}

func (constant *Constant) Constant() {}

func (constant *Constant) Type() types.Type {
	return constant._type
}

func (constant *Constant) Name() string { return constant.name }

func (constant *Constant) QualifiedName() string {
	if constant.Namespace != nil {
		return constant.Namespace.QualifiedName() + constant.name
	}
	return constant.name
}

func (constant *Constant) String() string {
	return fmt.Sprintf("%s = %s", constant.name, constant.Val)
}

type ConstantScope interface {
	Scope
	AddConstant(*Constant)
	ConstGet(string) (Const, error)
}

type Namespace interface {
	QualifiedName() string
}

type Module struct {
	name       string
	Statements Statements
	MethodSet  *MethodSet
	_type      types.Type
	lineNo     int
	Parent     *Module
	Constants  []*Constant
	Modules    []*Module
	Classes    []*Class
}

func (mod *Module) String() string {
	var body []string

	methods := []string{}
	for _, name := range mod.MethodSet.Order {
		m := mod.MethodSet.Methods[name]
		methods = append(methods, m.String())
	}
	if len(mod.Constants) > 0 {
		body = append(body, fmt.Sprintf("[%s]", stdlib.Join[*Constant](mod.Constants, "; ")))
	}
	if len(mod.Classes) > 0 {
		body = append(body, stdlib.Join[*Class](mod.Classes, "; "))
	}

	return fmt.Sprintf("%s(%s)", mod.name, Indent(body...))
}

func (mod *Module) Type() types.Type     { return mod._type }
func (mod *Module) SetType(t types.Type) { mod._type = t }
func (mod *Module) LineNo() int          { return mod.lineNo }
func (mod *Module) Copy() Node           { return mod }

func (mod *Module) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	GetType(mod.Statements, scope, nil)
	return nil, nil
}

func (mod *Module) Name() string {
	return mod.name
}

func (mod *Module) QualifiedName() string {
	if mod.Parent != nil {
		return mod.Parent.QualifiedName() + mod.name
	}
	return mod.name
}

func (mod *Module) Constant() {}

func (mod *Module) TakesConstants() bool {
	return true
}

func (mod *Module) AddConstant(constant *Constant) {
	constant.Namespace = mod
	mod.Constants = append(mod.Constants, constant)
}

func (mod *Module) ConstGet(name string) (Const, error) {
	for _, constant := range mod.Constants {
		if constant.Name() == name {
			return constant, nil
		}
	}
	for _, submod := range mod.Modules {
		if submod.Name() == name {
			return submod, nil
		}
	}
	for _, cls := range mod.Classes {
		if cls.Name() == name {
			return cls, nil
		}
	}
	return nil, fmt.Errorf("Module '%s' has no module, class, or constant '%s'", mod.Name(), name)
}

func (mod *Module) Get(name string) (Local, bool) {
	for _, constant := range mod.Constants {
		if constant.Name() == name {
			return constant, true
		}
	}
	return BadLocal, false
}

func (mod *Module) Set(string, Local) {}

type Class struct {
	name, Superclass string
	Statements       Statements
	MethodSet        *MethodSet
	_type            types.Type
	lineNo           int
	Body             Body
	ivars            map[string]*IVar
	ivarOrder        []string
	Constants        []*Constant
	Module           *Module
	Private          bool
}

func (cls *Class) String() string {
	var body []string

	if len(cls.Constants) > 0 {
		body = append(body, fmt.Sprintf("[%s]", stdlib.Join[*Constant](cls.Constants, "; ")))
	}

	instanceVars := []string{}
	for _, ivar := range cls.IVars(nil) {
		name := "@" + ivar.Name
		switch {
		case ivar.Readable && ivar.Writeable:
			name += "+rw"
		case ivar.Readable:
			name += "+r"
		case ivar.Writeable:
			name += "+w"
		}
		instanceVars = append(instanceVars, name)
	}
	if len(instanceVars) > 0 {
		body = append(body, fmt.Sprintf("{%s}", strings.Join(instanceVars, ", ")))
	}

	methods := []string{}
	for _, name := range cls.MethodSet.Order {
		m := cls.MethodSet.Methods[name]
		methods = append(methods, m.String())
	}
	if len(methods) > 0 {
		body = append(body, strings.Join(methods, "; "))
	}
	return fmt.Sprintf("%s(%s)", cls.Name(), Indent(body...))
}

func (cls *Class) Type() types.Type     { return cls._type }
func (cls *Class) SetType(t types.Type) { cls._type = t }
func (cls *Class) LineNo() int          { return cls.lineNo }

func (cls *Class) BuildType(outerScope ScopeChain) *types.Class {
	super := "Object"
	if cls.Superclass != "" {
		super = cls.Superclass
	}
	class := types.NewClass(cls.name, super, nil, types.ClassRegistry)
	class.Prefix = outerScope.Prefix()
	class.UserDefined = true
	for _, name := range cls.MethodSet.Order {
		m := cls.MethodSet.Methods[name]
		cls.GenerateMethod(m, class)
	}

	class.Instance.Def("initialize", types.MethodSpec{
		ReturnType: func(receiverType types.Type, blockReturnType types.Type, args []types.Type) (types.Type, error) {
			return class.Instance.(types.Type), nil
		},
		//blockArgs    func(Type, []Type) []Type
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			return types.Transform{
				Expr: bst.Call(nil, class.Constructor(), types.UnwrapTypeExprs(args)...),
			}
		},
	})
	cls._type = class

	// Where there's an implicit receiver, we may have too aggressively assigned
	// the call and the method doesn't even exist here.
	for name, list := range cls.MethodSet.Calls {
		if _, defined := cls.MethodSet.Methods[name]; !defined {
			for _, call := range list {
				if _, ok := call.Receiver.(*SelfNode); ok {
					call.Receiver = nil
					globalMethodSet.AddCall(call)
				}
			}
		}
	}

	GetType(cls.Statements, outerScope.Extend(cls), cls)

	return class
}

func (cls *Class) GenerateMethod(m *Method, class *types.Class) {
	// insert the class as a scope immediately after the method's locals
	m.Scope = append(m.Scope[:len(m.Scope)-1], ScopeChain{cls, m.Scope[len(m.Scope)-1]}...)
	// track internal calls to own methods here where receiver is implicit
	for _, c := range cls.MethodSet.Calls[m.Name] {
		c.Receiver = &SelfNode{_type: class.Instance.(types.Type), lineNo: c.lineNo}
	}
	class.Instance.Def(m.Name, types.MethodSpec{
		ReturnType: func(receiverType types.Type, blockReturnType types.Type, args []types.Type) (types.Type, error) {
			return m.ReturnType(), nil
		},

		//blockArgs    func(Type, []Type) []Type
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			return types.Transform{
				Expr: bst.Call(rcvr.Expr, m.GoName(), types.UnwrapTypeExprs(args)...),
			}
		},
	})
}

func (cls *Class) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	return cls._type, nil
}

func (cls *Class) Copy() Node {
	return cls
}

func (cls *Class) AddStatement(stmt Node) {
	if _, ok := stmt.(*Method); !ok {
		cls.Statements = append(cls.Statements, stmt)
	}
}

//ClassNode implements Scope with these methods
func (cls *Class) Get(name string) (Local, bool) {
	if ivar, ok := cls.ivars[name]; ok && ivar.Readable {
		return ivar, true
	} else if m, ok := cls.MethodSet.Methods[name]; ok && len(m.Params) == 0 {
		classType, _ := types.ClassRegistry.Get(cls.name)
		call := &MethodCall{
			Receiver:   &SelfNode{_type: classType.Instance.(types.Type)},
			Method:     m,
			MethodName: m.Name,
			_type:      m.ReturnType(),
		}
		GetType(call, ScopeChain{cls}, cls)
		return call, true
	}
	for _, constant := range cls.Constants {
		if constant.Name() == name {
			return constant, true
		}
	}
	return BadLocal, false
}

func (cls *Class) Set(string, Local) {}

func (cls *Class) Name() string {
	return cls.name
}

func (cls *Class) QualifiedName() string {
	if cls.Module != nil {
		return cls.Module.QualifiedName() + cls.name
	}
	return cls.name
}

func (cls *Class) TakesConstants() bool {
	return true
}

func (cls *Class) AddConstant(constant *Constant) {
	constant.Namespace = cls
	cls.Constants = append(cls.Constants, constant)
}

func (cls *Class) Parent() *Class {
	if cls.Superclass == "" {
		return nil
	}
	parentType, err := types.ClassRegistry.Get(cls.Superclass)
	if err != nil {
		panic(err)
	}
	if parent, ok := classMethodSets[parentType.Instance.(types.Type)]; ok {
		return parent.Class
	}
	return nil
}

func (cls *Class) Methods(skip map[string]bool) []*Method {
	if skip == nil {
		skip = map[string]bool{}
	}
	methods := []*Method{}
	for _, name := range cls.MethodSet.Order {
		if _, ok := skip[name]; !ok {
			m := cls.MethodSet.Methods[name]
			methods = append(methods, m)
			skip[name] = true
		}
	}
	if cls.Parent() != nil {
		methods = append(methods, cls.Parent().Methods(skip)...)
	}
	return methods
}

func (cls *Class) GetAncestorMethod(methodName string) (*Class, *Method, bool) {
	parent := cls.Parent()
	for parent != nil {
		if m, ok := parent.MethodSet.Methods[methodName]; ok {
			return parent, m, ok
		}
		parent = parent.Parent()
	}
	return nil, nil, false
}

func (cls *Class) AddIVars(args ArgsNode, readable, writeable bool) {
	for _, a := range args {
		sym, ok := a.(*SymbolNode)
		if ok {
			//TODO this method needs to return an error
			name := strings.TrimLeft(sym.Val, ":")
			ivar := &IVar{Name: name, Readable: readable, Writeable: writeable}
			cls.AddIVar(name, ivar)
		}
	}
}
func (cls *Class) AddIVar(name string, ivar *IVar) error {
	if existing, ok := cls.ivars[name]; ok {
		if existing.Type() != ivar.Type() {
			return fmt.Errorf("Attempted to set @%s on %s with %s but already was assigned %s", name, cls.Name(), ivar.Type(), existing.Type())
		} else if existing.Readable == ivar.Readable && existing.Writeable == ivar.Writeable {
			return nil
		} else if ivar.Readable || ivar.Writeable {
			existing.Readable, existing.Writeable = ivar.Readable, ivar.Writeable
			ivar = existing
		}
	} else {
		cls.ivars[name] = ivar
		cls.ivarOrder = append(cls.ivarOrder, name)
	}
	if ivar.Readable && !ivar.Writeable {
		scope := NewScope(ivar.Name + "Get")
		method := &Method{
			Name:      name,
			Locals:    scope,
			Scope:     ScopeChain{scope},
			ParamList: NewParamList(),
			Body: &Body{
				Statements: Statements{
					&IVarNode{
						Val:   "@" + ivar.Name,
						Class: cls,
						_type: ivar.Type(),
					},
				},
				ReturnType: ivar.Type(),
			},
		}
		cls.MethodSet.AddMethod(method)
		cls.GenerateMethod(method, cls.Type().(*types.Class))
	}
	if !ivar.Readable && ivar.Writeable {
		scope := NewScope(ivar.Name + "Set")
		paramList := NewParamList()
		paramList.AddParam(&Param{
			Name:  name,
			_type: ivar.Type(),
		})
		method := &Method{
			Name:      name + "=",
			Locals:    scope,
			Scope:     ScopeChain{scope},
			ParamList: paramList,
			Body: &Body{
				Statements: Statements{
					&AssignmentNode{
						Left: []Node{
							&IVarNode{
								Val:   "@" + ivar.Name,
								Class: cls,
								_type: ivar.Type(),
							},
						},
						Right: []Node{
							&IdentNode{
								Val: name,
							},
						},
						Reassignment: true,
					},
				},
				ReturnType: ivar.Type(),
			},
		}
		cls.MethodSet.AddMethod(method)
		cls.GenerateMethod(method, cls.Type().(*types.Class))
	}
	return nil
}

func (cls *Class) IVars(skip map[string]bool) []*IVar {
	ivars := []*IVar{}
	if skip == nil {
		skip = map[string]bool{}
	}
	for _, name := range cls.ivarOrder {
		if _, ok := skip[name]; !ok {
			ivar := cls.ivars[name]
			ivar.Name = name
			ivars = append(ivars, ivar)
			skip[name] = true
		}
	}
	if cls.Parent() != nil {
		ivars = append(ivars, cls.Parent().IVars(skip)...)
	}
	return ivars
}

func (cls *Class) Constant() {}

func (cls *Class) ConstGet(name string) (Const, error) {
	for _, constant := range cls.Constants {
		if constant.Name() == name {
			return constant, nil
		}
	}
	return nil, fmt.Errorf("Class '%s' has no constant '%s'", cls.Name(), name)
}

type ScopeAccessNode struct {
	Receiver Node
	Constant string
	_type    types.Type
	lineNo   int
}

func (n *ScopeAccessNode) String() string {
	return fmt.Sprintf("(%s::%s)", n.ReceiverName(), n.Constant)
}
func (n *ScopeAccessNode) Type() types.Type     { return n._type }
func (n *ScopeAccessNode) SetType(t types.Type) { n._type = t }
func (n *ScopeAccessNode) LineNo() int          { return n.lineNo }

func (n *ScopeAccessNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if constant, err := n.Walk(locals); err != nil {
		return nil, err
	} else {
		return constant.Type(), nil
	}
}

func (n *ScopeAccessNode) Lookup(scope ScopeChain, outer, inner string) (Const, error) {
	constant := scope.ResolveVar(outer)
	if constant == BadLocal {
		return nil, NewParseError(n, "No such class or module '%s'", outer)
	}
	if realConst, err := constant.(ConstantScope).ConstGet(inner); err != nil {
		return nil, NewParseError(n, err.Error())
	} else {
		return realConst, nil
	}
}

func (n *ScopeAccessNode) Walk(scope ScopeChain) (Const, error) {
	//base case -- not a scope chain
	if node, ok := n.Receiver.(*ScopeAccessNode); !ok {
		return n.Lookup(scope, n.ReceiverName(), n.Constant)
	} else {
		outerConstant, err := node.Walk(scope)
		if err != nil {
			return nil, err
		}
		if realConst, err := outerConstant.(ConstantScope).ConstGet(n.Constant); err != nil {
			return nil, NewParseError(n, err.Error())
		} else {
			return realConst, nil
		}
	}
}

func (n *ScopeAccessNode) Copy() Node {
	return &ScopeAccessNode{n.Receiver.Copy(), n.Constant, n._type, n.lineNo}
}

func (n *ScopeAccessNode) ReceiverName() string {
	switch node := n.Receiver.(type) {
	case *ConstantNode:
		return node.Val
	case *IdentNode:
		return node.Val
	case *Module:
		return node.Name()
	default:
		panic(NewParseError(n, "Scope operator (::) used on type other than a possible class/module. While technically valid Ruby, nobody really does this and the grammar shouldn't allow it."))

	}
}

type SuperNode struct {
	Args   ArgsNode
	Method *Method
	Class  *Class
	_type  types.Type
	lineNo int
}

func (n *SuperNode) String() string {
	params := n.Method.paramString()
	if len(n.Args) > 0 {
		params = stdlib.Join[Node](n.Args, ", ")
	}
	return fmt.Sprintf("super(%s)", params)
}
func (n *SuperNode) Type() types.Type     { return n._type }
func (n *SuperNode) SetType(t types.Type) { n._type = t }
func (n *SuperNode) LineNo() int          { return n.lineNo }

func (n *SuperNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	ancestor, method, found := n.Class.GetAncestorMethod(n.Method.Name)
	if !found {
		return nil, NewParseError(n, "Called super inside %s#%s but no ancestors have instance method %s", n.Class.Name(), n.Method.Name, n.Method.Name)
	}
	args := n.Args
	if len(args) == 0 && len(method.Params) > 0 {
		for _, param := range method.Params {
			loc, found := locals.Get(param.Name)
			if !found {
				return nil, NewParseError(n, "Detected mismatch in signatures of %s#%s and %s#%s, so cannot use bare super", ancestor.Name(), method.Name, n.Class.Name(), method.Name)
			}
			method.Locals.Set(param.Name, &RubyLocal{_type: loc.Type()})
			args = append(args, &IdentNode{Val: param.Name, _type: loc.Type()})
		}
	}
	if len(args) > 0 {
		for i, arg := range n.Args {
			var param *Param
			if kv, ok := arg.(*KeyValuePair); ok {
				param = method.ParamList.GetParamByName(kv.Label)
				if param == nil {
					return nil, NewParseError(arg, "Gave keyword argument '%s' to super but %s#%s has no corresponding keyword argument", kv.Label, ancestor.Name(), method.Name)
				}
			} else {
				var err error
				param, err = method.ParamList.GetParam(i)
				if err != nil {
					return nil, NewParseError(arg, "Gave positional argument '%s' to super but %s#%s has no corresponding positional argument", arg, ancestor.Name(), method.Name)
				}
			}
		}
	}
	superCall := &MethodCall{
		Receiver:   &SelfNode{_type: ancestor.Type().(*types.Class).Instance.(types.Type)},
		Method:     method,
		MethodName: method.Name,
		Args:       args,
		lineNo:     n.lineNo,
	}
	return GetType(superCall, locals, class)
}

func (n *SuperNode) Copy() Node {
	return &SuperNode{n.Args.Copy().(ArgsNode), n.Method, n.Class, n._type, n.lineNo}
}

func (n *SuperNode) Inline() Statements {
	_, method, _ := n.Class.GetAncestorMethod(n.Method.Name)
	return method.Body.Statements.Copy().(Statements)
}

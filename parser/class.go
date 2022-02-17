package parser

import (
	"fmt"
	"strings"

	"github.com/redneckbeard/thanos/bst"
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
	if n.Class != nil {
		if spec, exists := n.Class.ivars[n.NormalizedVal()]; exists {
			return spec.Type(), nil
		}
	}
	return nil, NewParseError(n, "Unable to connect to the mothership")
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

type IVar struct {
	Name                string
	_type               types.Type
	Readable, Writeable bool
}

func (ivar *IVar) Type() types.Type {
	return ivar._type
}

type Class struct {
	name, Superclass string
	Statements       Statements
	MethodSet        *MethodSet
	_type            types.Type
	lineNo           int
	Body             Body
	ivars            map[string]*IVar
	ivarOrder        []string
	Private          bool
}

func (cls *Class) String() string {
	methods := []string{}
	for _, name := range cls.MethodSet.Order {
		m := cls.MethodSet.Methods[name]
		methods = append(methods, m.String())
	}
	instanceVars := []string{}
	for _, name := range cls.ivarOrder {
		ivar := cls.ivars[name]
		name = "@" + name
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
	return fmt.Sprintf("%s([%s] %s)", cls.Name(), strings.Join(instanceVars, ", "), strings.Join(methods, "; "))
}

func (cls *Class) Type() types.Type     { return cls._type }
func (cls *Class) SetType(t types.Type) { cls._type = t }
func (cls *Class) LineNo() int          { return cls.lineNo }

func (cls *Class) BuildType() *types.Class {
	super := "Object"
	if cls.Superclass != "" {
		super = cls.Superclass
	}
	class := types.NewClass(cls.name, super, nil, types.ClassRegistry)
	class.UserDefined = true
	for _, name := range cls.MethodSet.Order {
		m := cls.MethodSet.Methods[name]
		// insert the class as a scope immediately after the method's locals
		m.Scope = append(m.Scope[:len(m.Scope)-1], ScopeChain{cls, m.Scope[len(m.Scope)-1]}...)
		// track internal calls to own methods here where receiver is implicit
		for _, c := range cls.MethodSet.Calls[m.Name] {
			c.Receiver = &SelfNode{_type: class.Instance.(types.Type), lineNo: c.lineNo}
		}
		class.Instance.Def(m.Name, func(m *Method) types.MethodSpec {
			return types.MethodSpec{
				ReturnType: func(receiverType types.Type, blockReturnType types.Type, args []types.Type) (types.Type, error) {
					return m.ReturnType(), nil
				},

				//blockArgs    func(Type, []Type) []Type
				TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
					name := m.Name
					if !m.Private {
						name = strings.Title(m.Name)
					}
					return types.Transform{
						Expr: bst.Call(rcvr.Expr, name, types.UnwrapTypeExprs(args)...),
					}
				},
			}
		}(m))
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

	for _, stmt := range cls.Statements {
		if c, ok := stmt.(*MethodCall); ok {
			switch c.MethodName {
			case "attr_reader":
				cls.AddIVars(c.Args, true, false)
			case "attr_writer":
				cls.AddIVars(c.Args, false, true)
			case "attr_accessor":
				cls.AddIVars(c.Args, true, true)
			}
		}
	}
	for _, name := range []string{"attr_reader", "attr_writer", "attr_accessor"} {
		delete(cls.MethodSet.Calls, name)
	}

	return class
}

func (cls *Class) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	return cls._type, nil
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
		return call, true
	}
	return BadLocal, false
}

func (cls *Class) Set(string, Local) {}

func (cls *Class) Name() string {
	return cls.name
}

func (cls *Class) IsClass() bool {
	return true
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

func (cls *Class) AddIVars(args ArgsNode, readable, writeable bool) {
	for _, a := range args {
		sym, ok := a.(*SymbolNode)
		if ok {
			//TODO this method needs to return an error
			name := strings.TrimLeft(sym.Val, ":")
			cls.ivars[name] = &IVar{Readable: readable, Writeable: writeable}
			cls.ivarOrder = append(cls.ivarOrder, name)
		}
	}
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

package types

import (
	"fmt"
	"go/ast"
	"reflect"
	"sort"
	"sync"

	"github.com/redneckbeard/thanos/bst"
)

type Class struct {
	name, parentName, Prefix string
	*proto
	Instance    instance
	parent      *Class
	children    []*Class
	UserDefined bool
}

func NewClass(name, parent string, inst instance, registry *classRegistry) *Class {
	class := &Class{
		name:       name,
		parentName: parent,
		Instance:   inst,
		proto:      newProto(name, parent, registry),
	}
	if inst == nil {
		class.Instance = Instance{name: name, proto: newProto(name, parent, registry)}
	}
	registry.RegisterClass(class)
	return class
}

var classProto *proto = newProto("Class", "Object", ClassRegistry)

func (t *Class) Equals(t2 Type) bool { return reflect.DeepEqual(t, t2) }
func (t *Class) String() string      { return fmt.Sprintf("%sClass", t.name) }
func (t *Class) GoType() string      { return t.Prefix + t.name }
func (t *Class) IsComposite() bool   { return false }

func (t *Class) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.MustResolve(m).ReturnType(t, b, args)
}

func (t *Class) BlockArgTypes(m string, args []Type) []Type {
	return t.MustResolve(m).blockArgs(t, args)
}

func (t *Class) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t *Class) Resolve(m string) (MethodSpec, bool) {
	if m == "new" {
		return t.Instance.Resolve("initialize")
	}
	return t.proto.Resolve(m, true)
}

func (t *Class) MustResolve(m string) MethodSpec {
	method, has := t.Resolve(m)
	if !has {
		panic(fmt.Errorf("Could not resolve class method '%s' on class '%s'", m, t))
	}
	return method
}

func (t *Class) HasMethod(m string) bool {
	_, has := t.Resolve(m)
	return has
}

func (t *Class) Constructor() string {
	return fmt.Sprintf("New%s", t.Prefix+t.name)
}

type classRegistry struct {
	sync.Mutex
	sync.WaitGroup
	registry    map[string]*Class
	initialized bool
}

var ClassRegistry = &classRegistry{registry: make(map[string]*Class)}

func (cr *classRegistry) Get(name string) (*Class, error) {
	if !cr.initialized {
		return nil, fmt.Errorf("Attempted to look up class '%s' from registry before registry initialized", name)
	}
	if class, found := cr.registry[name]; found {
		return class, nil
	}
	return nil, fmt.Errorf("Attempted to look up class '%s' from registry but no matching class found", name)
}

func (cr *classRegistry) MustGet(name string) *Class {
	if class, found := cr.registry[name]; found {
		return class
	}
	panic(fmt.Sprintf("Failed to find class %s", name))
}

func (cr *classRegistry) RegisterClass(cls *Class) {
	cr.Lock()
	cr.registry[cls.name] = cls
	cr.Unlock()
	go func() {
		for cls.parent == nil && cls.parentName != "" {
			//TODO probably use a WaitGroup instead so Analyze can be guaranteed that all ancestry chains are established
			cr.Lock()
			parent, found := cr.registry[cls.parentName]
			cr.Unlock()
			if found {
				cls.parent = parent
				parent.children = append(parent.children, cls)
			}
		}
	}()
}

func (cr *classRegistry) Initialize() error {
	cr.Wait()
	for _, class := range cr.registry {
		if class.parentName != "" && class.parent == nil {
			parent, found := cr.registry[class.parentName]
			if !found {
				return fmt.Errorf("Class '%s' described as having parent '%s' but no class '%s' was ever registered", class.name, class.parentName, class.parentName)
			}
			class.parent = parent
			parent.children = append(parent.children, class)
		}
	}
	cr.initialized = true
	return nil
}

func (cr *classRegistry) Names() []string {
	names := []string{}
	for k := range cr.registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

type instance interface {
	Def(string, MethodSpec)
	Resolve(string) (MethodSpec, bool)
	MustResolve(string) MethodSpec
	HasMethod(string) bool
	GenerateMethods(interface{}, ...string)
	Methods() map[string]MethodSpec
	Alias(string, string)
}

type Instance struct {
	name string
	*proto
}

func (t Instance) Equals(t2 Type) bool { return reflect.DeepEqual(t, t2) }
func (t Instance) String() string      { return t.name }
func (t Instance) GoType() string      { return "*" + t.name }
func (t Instance) IsComposite() bool   { return false }

func (t Instance) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Instance) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Instance) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Instance) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Instance) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Instance) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Instance) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

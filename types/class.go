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
	Package                  string // Go package name (e.g., "animals"), empty for main
	PackagePath              string // Full import path (e.g., "tmpmod/animals")
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
		inst := Instance{name: name, proto: newProto(name, parent, registry)}
		inst.class = class
		class.Instance = inst
	}
	registry.RegisterClass(class)
	return class
}

var classProto *proto = newProto("Class", "Object", ClassRegistry)

func (t *Class) Equals(t2 Type) bool { return reflect.DeepEqual(t, t2) }
func (t *Class) String() string      { return fmt.Sprintf("%sClass", t.name) }
func (t *Class) GoType() string {
	if t.Package != "" {
		return t.name // no prefix when class lives in its own package
	}
	return t.Prefix + t.name
}
func (t *Class) IsComposite() bool   { return false }

func (t *Class) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.MustResolve(m).ReturnType(t, b, args)
}

func (t *Class) BlockArgTypes(m string, args []Type) []Type {
	spec := t.MustResolve(m)
	return spec.BlockArgs(t, args)
}

func (t *Class) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t *Class) Resolve(m string) (MethodSpec, bool) {
	if m == "new" {
		if spec, ok := t.Instance.Resolve("initialize"); ok {
			return spec, ok
		}
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

func (t *Class) GetMethodSpec(m string) (MethodSpec, bool) {
	return t.Resolve(m)
}

// ExternalGoType returns the package-qualified Go type name for cross-package references.
func (t *Class) ExternalGoType() string {
	if t.Package != "" {
		return t.Package + "." + t.name
	}
	return t.GoType()
}

// ExternalConstructor returns the package-qualified constructor name.
func (t *Class) ExternalConstructor() string {
	if t.Package != "" {
		return t.Package + ".New" + t.name
	}
	return t.Constructor()
}

func (t *Class) Constructor() string {
	if t.Package != "" {
		return "New" + t.name
	}
	return fmt.Sprintf("New%s", t.Prefix+t.name)
}

// Def registers a class-level method spec. Exported for use from external packages.
func (t *Class) Def(m string, spec MethodSpec) {
	t.proto.Def(m, spec)
}

// MakeAlias creates a method alias. Exported for use from external packages.
func (t *Class) MakeAlias(existingMethod, newMethod string, classMethod bool) {
	t.proto.MakeAlias(existingMethod, newMethod, classMethod)
}

// SetBlockArgs sets the blockArgs function for a class-level method spec.
// Needed because blockArgs is unexported on MethodSpec.
func (t *Class) SetBlockArgs(m string, f func(Type, []Type) []Type) {
	if spec, ok := t.proto.methods[m]; ok {
		spec.SetBlockArgs(f)
		t.proto.methods[m] = spec
	}
}

type classRegistry struct {
	sync.Mutex
	sync.WaitGroup
	registry    map[string]*Class
	builtins    map[string]bool
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
	// On first initialization, snapshot built-in class names.
	if cr.builtins == nil {
		cr.builtins = make(map[string]bool, len(cr.registry))
		for name := range cr.registry {
			cr.builtins[name] = true
		}
	}
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

// Reset removes all user-defined classes, keeping only built-in ones.
// Used between test runs to prevent state leakage.
func (cr *classRegistry) Reset() {
	if cr.builtins == nil {
		return
	}
	cr.Lock()
	defer cr.Unlock()
	for name := range cr.registry {
		if !cr.builtins[name] {
			delete(cr.registry, name)
		}
	}
	// Clear parent-child links that reference user-defined classes
	for _, cls := range cr.registry {
		cls.children = nil
	}
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
	name  string
	class *Class // back-reference for package info
	*proto
}

func (t Instance) Equals(t2 Type) bool { return reflect.DeepEqual(t, t2) }
func (t Instance) String() string      { return t.name }
func (t Instance) GoType() string { return "*" + t.name }

// ExternalGoType returns the package-qualified type for cross-package references.
func (t Instance) ExternalGoType() string {
	if t.class != nil && t.class.Package != "" {
		return "*" + t.class.Package + "." + t.name
	}
	return t.GoType()
}

func (t Instance) IsComposite() bool { return false }

func (t Instance) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Instance) BlockArgTypes(m string, args []Type) []Type {
	spec := t.proto.MustResolve(m, false)
	return spec.BlockArgs(t, args)
}

func (t Instance) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Instance) GetMethodSpec(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
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

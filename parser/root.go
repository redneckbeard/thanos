package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strings"

	"github.com/redneckbeard/thanos/bst"
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
	ModulePath       string // Go module path for imports (e.g., "tmpmod")
	currentClass     *Class
	currentMethod    *Method
	inPrivateMethods bool
	inSingletonClass    bool
	singletonTargetDepth int
	nextConstantType    int
	cpathDepth          int
	facades             types.FacadeConfig
	loadPaths           []string
	loadingGem          bool // true while parsing gem source files
}

func NewRoot() *Root {
	globalMethodSet = NewMethodSet()
	classMethodSets = make(map[types.Type]*MethodSet)
	ResetGlobalVars()
	ResetDuckInterfaces()
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
	if r.loadingGem {
		return
	}
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
	if f := p.node.File(); f != "" {
		return fmt.Sprintf("%s line %d: %s", f, p.node.LineNo(), p.msg)
	}
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

// PushSingletonTarget resolves an existing module or class by name and pushes
// it onto the module stack for `class << SomeConstant` blocks.
// For chained access like `class << Outer::Inner`, this is called once per segment.
// Each segment pushes onto the module stack so the next segment resolves within it.
func (r *Root) PushSingletonTarget(name string) {
	r.singletonTargetDepth++
	// findModule/findClass searches relative to moduleStack.Peek(),
	// so chained access like Outer::Inner works naturally.
	r.State.Push(InModuleBody)
	if existing := r.findModule(name); existing != nil {
		r.MethodSetStack.Push(existing.MethodSet)
		r.moduleStack.Push(existing)
		r.ScopeChain = r.ScopeChain.Extend(existing)
		return
	}
	if existing := r.findClass(name); existing != nil {
		r.MethodSetStack.Push(existing.MethodSet)
		r.moduleStack.Push(&Module{name: name, MethodSet: existing.MethodSet})
		r.ScopeChain = r.ScopeChain.Extend(existing)
		return
	}
	// Not found — create a placeholder module
	mod := &Module{name: name, Pos: Pos{lineNo: 0}}
	ms := NewMethodSet()
	mod.MethodSet = ms
	ms.Module = mod
	r.MethodSetStack.Push(ms)
	r.moduleStack.Push(mod)
	r.ScopeChain = r.ScopeChain.Extend(mod)
}

// PopSingletonTarget pops the module/class pushed by PushSingletonTarget.
// Re-registers any new class methods added during the singleton block.
func (r *Root) PopSingletonTarget() {
	// The target module is at the top of the stack.
	// If there were intermediate segments (e.g., Outer in Outer::Inner),
	// they're below and need to be popped too.
	module := r.moduleStack.Pop()
	r.MethodSetStack.Pop()
	r.State.Pop()

	// Pop intermediate segments (all except the last one which is the target)
	for i := 1; i < r.singletonTargetDepth; i++ {
		r.moduleStack.Pop()
		r.MethodSetStack.Pop()
		r.State.Pop()
		r.ScopeChain = r.ScopeChain[:len(r.ScopeChain)-1]
	}

	// Re-register class methods on the module's type
	pkgName := strings.ToLower(module.name)
	if modClass, ok := module._type.(*types.Class); ok {
		for _, m := range module.ClassMethods {
			if modClass.HasMethod(m.Name) {
				continue
			}
			m.Scope = append(m.Scope[:len(m.Scope)-1], ScopeChain{module, m.Scope[len(m.Scope)-1]}...)
			funcName := pkgName + "." + GoName(m.Name)
			modClass.Def(m.Name, generateClassMethodSpec(m, funcName, modClass, nil, false))
		}
	} else if len(module.ClassMethods) > 0 {
		modClass := types.NewClass(module.name, "Object", nil, types.ClassRegistry)
		modClass.UserDefined = true
		modClass.Package = pkgName
		for _, m := range module.ClassMethods {
			m.Scope = append(m.Scope[:len(m.Scope)-1], ScopeChain{module, m.Scope[len(m.Scope)-1]}...)
			funcName := pkgName + "." + GoName(m.Name)
			modClass.Def(m.Name, generateClassMethodSpec(m, funcName, modClass, nil, false))
		}
		module._type = modClass
	}

	GetType(module, r.ScopeChain, nil)
	r.ScopeChain = r.ScopeChain[:len(r.ScopeChain)-1]
	r.ScopeChain.Set(module.Name(), module)
	r.singletonTargetDepth = 0
}

// ConvertClassToModule converts the current class (pushed as the base of a cpath)
// back to a module. This is needed when `class Outer::Inner::Item` is parsed —
// the grammar initially pushes `Outer` as a class (because nextConstantType == CLASS),
// but once `::` is seen, we know `Outer` is just an intermediate module container.
func (r *Root) ConvertClassToModule() {
	name := r.currentClass.name
	lineNo := r.currentClass.lineNo
	r.MethodSetStack.Pop()
	r.currentClass = nil
	r.State.Pop()
	// Now push as a module instead
	r.PushModule(name, lineNo)
}

func (r *Root) PushModule(name string, lineNo int) {
	r.State.Push(InModuleBody)
	// Check for existing module (reopening)
	if existing := r.findModule(name); existing != nil {
		if r.loadingGem {
			existing.fromGem = true
		}
		r.MethodSetStack.Push(existing.MethodSet)
		r.moduleStack.Push(existing)
		r.ScopeChain = r.ScopeChain.Extend(existing)
		return
	}
	mod := &Module{name: name, Pos: Pos{lineNo: lineNo}, fromGem: r.loadingGem}
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
	r.State.Pop()

	// If the module has class methods (def self.x), create a type for resolution
	pkgName := strings.ToLower(module.name)
	if len(module.ClassMethods) > 0 || len(module.Classes) > 0 {
		modClass := types.NewClass(module.name, "Object", nil, types.ClassRegistry)
		modClass.UserDefined = true
		modClass.Package = pkgName
		for _, m := range module.ClassMethods {
			m.Scope = append(m.Scope[:len(m.Scope)-1], ScopeChain{module, m.Scope[len(m.Scope)-1]}...)
			funcName := pkgName + "." + GoName(m.Name)
			modClass.Def(m.Name, generateClassMethodSpec(m, funcName, modClass, nil, m.FromGem))
		}
		module._type = modClass
	}

	GetType(module, r.ScopeChain, nil)
	r.ScopeChain = r.ScopeChain[:len(r.ScopeChain)-1]
	r.ScopeChain.Set(module.Name(), module)
	module.Parent = r.moduleStack.Peek()
	return module
}

// PopIntermediateModule pops a module that was pushed as part of a :: chain
// in a cpath (e.g., Diff and LCS in `module Diff::LCS::Internals`).
// These intermediate modules get registered in the tree but their body is empty.
func (r *Root) PopIntermediateModule() {
	module := r.moduleStack.Pop()
	r.MethodSetStack.Pop()
	r.State.Pop()

	// Run type inference on the module (registers class methods etc.)
	GetType(module, r.ScopeChain, nil)

	r.ScopeChain = r.ScopeChain[:len(r.ScopeChain)-1]
	// Register module name in parent scope for ScopeAccessNode resolution
	r.ScopeChain.Set(module.Name(), module)
	module.Parent = r.moduleStack.Peek()

	// Register in parent's Modules list if not already there (reopened modules are already registered)
	if parent := r.moduleStack.Peek(); parent != nil {
		found := false
		for _, m := range parent.Modules {
			if m == module {
				found = true
				break
			}
		}
		if !found {
			parent.Modules = append(parent.Modules, module)
		}
	} else {
		found := false
		for _, m := range r.TopLevelModules {
			if m == module {
				found = true
				break
			}
		}
		if !found {
			r.TopLevelModules = append(r.TopLevelModules, module)
		}
	}
}

func (r *Root) PushClass(name string, lineNo int) {
	r.State.Push(InClassBody)
	// Check if the class already exists (open class / monkey patching)
	if existing := r.findClass(name); existing != nil {
		r.currentClass = existing
		r.MethodSetStack.Push(existing.MethodSet)
		return
	}
	cls := &Class{name: name, Pos: Pos{lineNo: lineNo}, ivars: make(map[string]*IVar), cvars: make(map[string]*CVar)}
	ms := NewMethodSet()
	cls.MethodSet = ms
	ms.Class = cls
	r.currentClass = cls
	r.MethodSetStack.Push(ms)
}

func (r *Root) findClass(name string) *Class {
	if parent := r.moduleStack.Peek(); parent != nil {
		for _, cls := range parent.Classes {
			if cls.Name() == name {
				return cls
			}
		}
	} else {
		for _, cls := range r.Classes {
			if cls.Name() == name {
				return cls
			}
		}
	}
	return nil
}

func (r *Root) findModule(name string) *Module {
	if parent := r.moduleStack.Peek(); parent != nil {
		for _, mod := range parent.Modules {
			if mod.Name() == name {
				return mod
			}
		}
	} else {
		for _, mod := range r.TopLevelModules {
			if mod.Name() == name {
				return mod
			}
		}
	}
	return nil
}

func (r *Root) PopClass() *Class {
	class := r.currentClass
	r.MethodSetStack.Pop()
	r.currentClass = nil
	// Only add to class list if not already present (open class reopening)
	if r.findClass(class.Name()) == nil {
		if parent := r.moduleStack.Peek(); parent != nil {
			parent.Classes = append(parent.Classes, class)
		} else {
			r.Classes = append(r.Classes, class)
		}
	}
	r.inPrivateMethods = false
	class.Module = r.moduleStack.Peek()
	t := class.BuildType(r.ScopeChain)
	classMethodSets[t.Instance.(types.Type)] = class.MethodSet
	r.State.Pop()
	r.ScopeChain.Set(class.Name(), class)
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
	if m.ClassMethod && r.currentClass != nil {
		r.currentClass.ClassMethods = append(r.currentClass.ClassMethods, m)
	} else if m.ClassMethod && r.moduleStack.Peek() != nil {
		r.moduleStack.Peek().ClassMethods = append(r.moduleStack.Peek().ClassMethods, m)
	} else {
		ms := r.MethodSetStack.Peek()
		ms.AddMethod(m)
	}
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
		case *ScopeAccessNode:
			// Register the call in the target module/class's MethodSet so
			// that analyzeModuleClassMethods can populate param types later.
			r.registerScopedCall(rcvr, c)
		}
	} else if c.MethodName == "include" && r.currentClass != nil {
		// Handle `include` immediately during parsing so cls.Includes is
		// populated before BuildType runs in PopClass.
		if len(c.Args) == 1 {
			if constNode, ok := c.Args[0].(*ConstantNode); ok {
				r.currentClass.Includes = append(r.currentClass.Includes, constNode.Val)
			}
		}
		return
	} else {
		// Inside a class method body, check class methods on the current class
		// before instance methods. This ensures `def self.patch!` calling
		// `patch(...)` resolves to `def self.patch(...)` instead of `def patch(...)`.
		var method *Method
		var cls *Class
		if r.currentMethod != nil && r.currentMethod.ClassMethod {
			// Check current class's class methods
			if r.currentClass != nil {
				for _, m := range r.currentClass.ClassMethods {
					if m.Name == c.MethodName {
						method = m
						cls = r.currentClass
						break
					}
				}
			}
			// Check current module's class methods
			if method == nil && r.moduleStack.Peek() != nil {
				for _, m := range r.moduleStack.Peek().ClassMethods {
					if m.Name == c.MethodName {
						method = m
						break
					}
				}
			}
		}
		if method == nil {
			if m, found := r.MethodSetStack.Peek().Methods[c.MethodName]; found {
				method = m
				cls = r.MethodSetStack.Peek().Class
			}
		}
		if method != nil && !r.loadingGem {
			if err := method.AnalyzeArguments(cls, c, nil); err != nil {
				r.AddError(err)
				if e, ok := err.(*ParseError); ok && e.terminal {
					return
				}
			}
		}
	}
	if calls, ok := r.MethodSetStack.Peek().Calls[c.MethodName]; ok {
		r.MethodSetStack.Peek().Calls[c.MethodName] = append(calls, c)
	} else {
		r.MethodSetStack.Peek().Calls[c.MethodName] = []*MethodCall{c}
	}
}

// registerScopedCall registers a method call in the target module or class's
// MethodSet when the receiver is a ScopeAccessNode (e.g. Diff::LCS::Internals.lcs).
// This allows analyzeModuleClassMethods to populate param types from calls.
func (r *Root) registerScopedCall(rcvr *ScopeAccessNode, c *MethodCall) {
	var targetMod *Module
	var targetCls *Class
	func() {
		defer func() { recover() }()
		constant, err := rcvr.Walk(r.ScopeChain)
		if err != nil {
			if Tracer != nil {
				Tracer.Record("scoped-call-err", fmt.Sprintf("%s.%s: %v", rcvr, c.MethodName, err))
			}
			return
		}
		if Tracer != nil {
			Tracer.Record("scoped-call-resolved", fmt.Sprintf("%s.%s => %T", rcvr, c.MethodName, constant))
		}
		switch resolved := constant.(type) {
		case *Module:
			targetMod = resolved
		case *Class:
			targetCls = resolved
		}
	}()
	if targetMod != nil {
		targetMod.MethodSet.Calls[c.MethodName] = append(
			targetMod.MethodSet.Calls[c.MethodName], c)
		if !r.loadingGem {
			for _, m := range targetMod.ClassMethods {
				if m.Name == c.MethodName {
					m.AnalyzeArguments(nil, c, nil)
					break
				}
			}
		}
	} else if targetCls != nil {
		targetCls.MethodSet.Calls[c.MethodName] = append(
			targetCls.MethodSet.Calls[c.MethodName], c)
		if !r.loadingGem {
			for _, m := range targetCls.ClassMethods {
				if m.Name == c.MethodName {
					m.AnalyzeArguments(targetCls, c, nil)
					break
				}
			}
		}
	}
}

// resolveDeferredScopedCalls walks all recorded calls across every MethodSet
// looking for calls with ScopeAccessNode receivers that weren't resolved during
// parsing (because the target module/class wasn't loaded yet). Now that all
// modules are available, it resolves them and registers in the target's Calls map.
func (r *Root) resolveDeferredScopedCalls() {
	r.resolveScopedCallsInMethodSet(r.MethodSetStack.Peek())
	for _, cls := range r.Classes {
		r.resolveScopedCallsInMethodSet(cls.MethodSet)
	}
	for _, mod := range r.TopLevelModules {
		r.resolveScopedCallsInModule(mod)
	}
}

func (r *Root) resolveScopedCallsInModule(mod *Module) {
	r.resolveScopedCallsInMethodSet(mod.MethodSet)
	for _, cls := range mod.Classes {
		r.resolveScopedCallsInMethodSet(cls.MethodSet)
	}
	for _, sub := range mod.Modules {
		r.resolveScopedCallsInModule(sub)
	}
}

func (r *Root) resolveScopedCallsInMethodSet(ms *MethodSet) {
	for _, calls := range ms.Calls {
		for _, c := range calls {
			rcvr, ok := c.Receiver.(*ScopeAccessNode)
			if !ok {
				continue
			}
			var targetMod *Module
			var targetCls *Class
			func() {
				defer func() { recover() }()
				constant, err := rcvr.Walk(r.ScopeChain)
				if err != nil {
					return
				}
				switch resolved := constant.(type) {
				case *Module:
					targetMod = resolved
				case *Class:
					targetCls = resolved
				}
			}()
			if targetMod != nil {
				// Avoid duplicate registration
				if !r.callAlreadyRegistered(targetMod.MethodSet, c) {
					targetMod.MethodSet.Calls[c.MethodName] = append(
						targetMod.MethodSet.Calls[c.MethodName], c)
					if Tracer != nil {
						Tracer.Record("deferred-scoped-call", fmt.Sprintf("%s.%s => module %s", rcvr, c.MethodName, targetMod.name))
					}
				}
				// Eagerly populate param types on the target method
				for _, m := range targetMod.ClassMethods {
					if m.Name == c.MethodName {
						func() {
							defer func() { recover() }()
							m.AnalyzeArguments(nil, c, nil)
						}()
						break
					}
				}
			} else if targetCls != nil {
				if !r.callAlreadyRegistered(targetCls.MethodSet, c) {
					targetCls.MethodSet.Calls[c.MethodName] = append(
						targetCls.MethodSet.Calls[c.MethodName], c)
					if Tracer != nil {
						Tracer.Record("deferred-scoped-call", fmt.Sprintf("%s.%s => class %s", rcvr, c.MethodName, targetCls.name))
					}
				}
				for _, m := range targetCls.ClassMethods {
					if m.Name == c.MethodName {
						func() {
							defer func() { recover() }()
							m.AnalyzeArguments(targetCls, c, nil)
						}()
						break
					}
				}
			}
		}
	}
}

func (r *Root) callAlreadyRegistered(ms *MethodSet, c *MethodCall) bool {
	for _, existing := range ms.Calls[c.MethodName] {
		if existing == c {
			return true
		}
	}
	return false
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

func (r *Root) analyzeClassMethodBodies() {
	// Analyze class methods on top-level classes
	for _, cls := range r.Classes {
		Tracer.Record("enter-class", fmt.Sprintf("%s (classMethods=%d)", cls.name, len(cls.ClassMethods)))
		for _, m := range cls.ClassMethods {
			Tracer.Record("analyze-args", fmt.Sprintf("%s.%s (%d calls)", cls.name, m.Name, len(cls.MethodSet.Calls[m.Name])))
			for _, c := range cls.MethodSet.Calls[m.Name] {
				func() {
					defer func() { recover() }()
					m.AnalyzeArguments(cls, c, nil)
				}()
			}
			if m.Body.ReturnType == nil {
				Tracer.Record("infer-return", fmt.Sprintf("%s.%s (attempting)", cls.name, m.Name))
				m.Body.InferReturnType(m.Scope, cls)
			}
			retStr := "nil"
			if m.Body.ReturnType != nil {
				retStr = m.Body.ReturnType.String()
			}
			Tracer.Record("method-result", fmt.Sprintf("%s.%s => %s", cls.name, m.Name, retStr))
		}
	}
	// Analyze class methods in module classes (recursive)
	Tracer.Record("enter-modules", fmt.Sprintf("%d top-level modules", len(r.TopLevelModules)))
	for _, mod := range r.TopLevelModules {
		r.analyzeModuleClassMethods(mod)
	}
}

// analyzeModuleConstants registers constants defined in module bodies so
// they are available before the first pass resolves method bodies.
func (r *Root) analyzeModuleConstants(mod *Module, parentScope ScopeChain) {
	modScope := parentScope.Extend(mod)
	for _, stmt := range mod.Statements {
		if assign, ok := stmt.(*AssignmentNode); ok {
			if len(assign.Left) == 1 {
				if _, ok := assign.Left[0].(*ConstantNode); ok {
					GetType(assign, modScope, nil)
				}
			}
		}
	}
	for _, sub := range mod.Modules {
		r.analyzeModuleConstants(sub, modScope)
	}
}

func (r *Root) analyzeModule(mod *Module, parentScope ScopeChain) error {
	modScope := parentScope.Extend(mod)
	if len(mod.Statements) > 0 {
		// Filter out Method and NoopNode entries from module statements —
		// methods are analyzed via AnalyzeMethodSet, and analyzing them
		// here via GetType would fail on unresolved params.
		var stmts Statements
		for _, s := range mod.Statements {
			switch s.(type) {
			case *Method, *NoopNode:
				continue
			default:
				stmts = append(stmts, s)
			}
		}
		if len(stmts) > 0 {
			if _, err := GetType(stmts, modScope, nil); err != nil {
				if mod.fromGem {
					fmt.Fprintf(os.Stderr, "warning: module %s: %v (continuing)\n", mod.Name(), err)
				} else {
					return err
				}
			}
		}
	}
	for i := len(mod.Classes) - 1; i >= 0; i-- {
		cls := mod.Classes[i]
		if err := r.AnalyzeMethodSet(cls.MethodSet, cls.Type()); err != nil {
			return err
		}
	}
	for _, sub := range mod.Modules {
		if err := r.analyzeModule(sub, modScope); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) analyzeModuleClassMethods(mod *Module) {
	// Analyze class methods defined directly on the module (def self.x inside module).
	// Multi-pass analysis for module class methods. The first pass populates
	// params and infers return types. Subsequent passes re-analyze gem methods
	// whose bodies contain calls to sibling methods that may have gained return
	// types in a previous pass (e.g. lcs calling replace_next_larger).
	Tracer.Record("enter-module", fmt.Sprintf("%s (classMethods=%d, subModules=%d, classes=%d)", mod.name, len(mod.ClassMethods), len(mod.Modules), len(mod.Classes)))
	// Two-pass analysis for module class methods. Pass 0 analyzes all methods,
	// populating return types (tolerant mode for gems). Pass 1 re-analyzes gem
	// methods with full type resets so that calls to sibling methods (which now
	// have return types from pass 0) resolve correctly inside the body.
	for pass := 0; pass < 2; pass++ {
		Tracer.Record("pass", fmt.Sprintf("%s pass %d", mod.name, pass))
		for _, m := range mod.ClassMethods {
			// Populate param types from recorded calls (mirrors Method.Analyze).
			Tracer.Record("analyze-args", fmt.Sprintf("%s.%s (%d calls, pass %d)", mod.name, m.Name, len(mod.MethodSet.Calls[m.Name]), pass))
			for _, c := range mod.MethodSet.Calls[m.Name] {
				func() {
					defer func() { recover() }()
					m.AnalyzeArguments(nil, c, nil)
				}()
			}
			// Log param types after AnalyzeArguments
			for _, p := range m.Params {
				if p.Type() != nil {
					Tracer.Record("param-type", fmt.Sprintf("%s.%s param %s => %s", mod.name, m.Name, p.Name, p.Type()))
				}
			}
			if pass > 0 && m.FromGem && len(m.Body.Statements) > 0 {
				// Re-analyze all gem methods on pass 1: their bodies may have
				// stale cached types from an earlier analysis pass where sibling
				// methods weren't yet resolved.
				Tracer.Record("body-reset", fmt.Sprintf("%s.%s (pass %d, retType=%v, tolerant=%v)", mod.name, m.Name, pass, m.Body.ReturnType, m.Body.tolerantInfer))
				m.Body.ReturnType = nil
				m.Body.tolerantInfer = false
				m.Body.clearCachedTypes()
				newLocals := NewScope(m.Locals.Name())
				for _, p := range m.Params {
					if p.Type() != nil {
						newLocals.Set(p.Name, &RubyLocal{_type: p.Type()})
					}
				}
				m.Locals = newLocals
				m.Scope = m.Scope[:len(m.Scope)-1].Extend(newLocals)
			}
			if m.Body.ReturnType == nil && len(m.Body.Statements) > 0 {
				Tracer.Record("infer-return", fmt.Sprintf("%s.%s (attempting, pass %d)", mod.name, m.Name, pass))
				if m.FromGem {
					func() {
						defer func() { recover() }()
						if err := m.Body.InferReturnType(m.Scope, nil); err != nil {
							Tracer.Record("infer-fallback", fmt.Sprintf("%s.%s strict failed: %v", mod.name, m.Name, err))
							inferGemMethodReturnType(m)
						}
					}()
				} else {
					m.Body.InferReturnType(m.Scope, nil)
				}
			}
			// Log result after inference attempt
			retStr := "nil"
			if m.Body.ReturnType != nil {
				retStr = m.Body.ReturnType.String()
			}
			tolerantStr := ""
			if m.Body.tolerantInfer {
				tolerantStr = " [tolerant]"
			}
			Tracer.Record("method-result", fmt.Sprintf("%s.%s => %s%s", mod.name, m.Name, retStr, tolerantStr))
		}
	}
	// Analyze class methods on classes within the module
	for _, cls := range mod.Classes {
		Tracer.Record("enter-module-class", fmt.Sprintf("%s::%s (classMethods=%d)", mod.name, cls.name, len(cls.ClassMethods)))
		for _, m := range cls.ClassMethods {
			Tracer.Record("analyze-args", fmt.Sprintf("%s::%s.%s (%d calls)", mod.name, cls.name, m.Name, len(cls.MethodSet.Calls[m.Name])))
			for _, c := range cls.MethodSet.Calls[m.Name] {
				func() {
					defer func() { recover() }()
					m.AnalyzeArguments(cls, c, nil)
				}()
			}
			if m.Body.ReturnType == nil && len(m.Body.Statements) > 0 {
				Tracer.Record("infer-return", fmt.Sprintf("%s::%s.%s (attempting)", mod.name, cls.name, m.Name))
				if m.FromGem {
					func() {
						defer func() { recover() }()
						m.Body.InferReturnType(m.Scope, cls)
					}()
				} else {
					m.Body.InferReturnType(m.Scope, cls)
				}
			}
			retStr := "nil"
			if m.Body.ReturnType != nil {
				retStr = m.Body.ReturnType.String()
			}
			Tracer.Record("method-result", fmt.Sprintf("%s::%s.%s => %s", mod.name, cls.name, m.Name, retStr))
		}
	}
	for _, sub := range mod.Modules {
		r.analyzeModuleClassMethods(sub)
	}
}

// inferGemMethodReturnType tries to determine the return type of a gem method
// when full InferReturnType failed (e.g. due to mid-body type errors). It runs
// InferReturnType in tolerant mode which continues past mid-body errors.
func inferGemMethodReturnType(m *Method) {
	if m.Body == nil || len(m.Body.Statements) == 0 {
		return
	}
	tolerantGetType = true
	defer func() { tolerantGetType = false }()
	m.Body.InferReturnType(m.Scope, nil)
	if m.Body.ReturnType != nil {
		m.Body.tolerantInfer = true
		Tracer.Record("tolerant-result", fmt.Sprintf("%s => %s (errors skipped)", m.Name, m.Body.ReturnType))
	} else {
		Tracer.Record("tolerant-result", fmt.Sprintf("%s => nil (still failed)", m.Name))
	}
}

// tolerantGetType causes Statements.TargetType to skip errors on individual
// statements rather than aborting the entire method. Set to true only during
// gem method analysis.
var tolerantGetType bool

func (r *Root) preAnalyzeInitializers() {
	for _, class := range r.Classes {
		r.preAnalyzeClassInitializer(class)
	}
	for _, mod := range r.TopLevelModules {
		r.preAnalyzeModuleInitializers(mod)
	}
}

func (r *Root) preAnalyzeClassInitializer(cls *Class) {
	if initialize, ok := cls.MethodSet.Methods["initialize"]; ok && !initialize.analyzed {
		if err := initialize.Analyze(cls.MethodSet); err == nil {
			initialize.analyzed = true
		}
	}
}

func (r *Root) preAnalyzeModuleInitializers(mod *Module) {
	for _, cls := range mod.Classes {
		r.preAnalyzeClassInitializer(cls)
	}
	for _, sub := range mod.Modules {
		r.preAnalyzeModuleInitializers(sub)
	}
}

func (r *Root) AnalyzeMethodSet(ms *MethodSet, rcvr types.Type) error {
	var err error
	unanalyzedCount := len(ms.Methods)
	for unanalyzedCount > 0 {
		successes := 0
		if initialize, ok := ms.Methods["initialize"]; ok && !initialize.analyzed {
			owner := "global"
			if ms.Class != nil {
				owner = ms.Class.name
			}
			Tracer.Record("analyze-method", fmt.Sprintf("%s#initialize (%d calls)", owner, len(ms.Calls["initialize"])))
			err = initialize.Analyze(ms)
			if err == nil {
				initialize.analyzed = true
				successes++
			} else {
				Tracer.Record("error", err.Error())
			}
		}
		for _, name := range ms.Order {
			m := ms.Methods[name]
			if !m.analyzed {
				owner := "global"
				if ms.Class != nil {
					owner = ms.Class.name
				}
				Tracer.Record("analyze-method", fmt.Sprintf("%s#%s (%d calls)", owner, name, len(ms.Calls[name])))
				err = m.Analyze(ms)
				if err == nil {
					m.analyzed = true
					successes++
					retType := "nil"
					if m.ReturnType() != nil {
						retType = m.ReturnType().String()
					}
					Tracer.Record("method-typed", fmt.Sprintf("%s => %s", name, retType))
				} else {
					Tracer.Record("error", err.Error())
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

func (r *Root) expandStructDefinitions() {
	var remaining []Node
	for _, stmt := range r.Statements {
		assign, ok := stmt.(*AssignmentNode)
		if !ok || len(assign.Left) != 1 || len(assign.Right) != 1 {
			remaining = append(remaining, stmt)
			continue
		}
		constNode, isConst := assign.Left[0].(*ConstantNode)
		if !isConst {
			remaining = append(remaining, stmt)
			continue
		}
		call, isCall := assign.Right[0].(*MethodCall)
		if !isCall {
			remaining = append(remaining, stmt)
			continue
		}
		rcvr, isConstRcvr := call.Receiver.(*ConstantNode)
		isStructNew := isConstRcvr && rcvr.Val == "Struct" && call.MethodName == "new"
		isDataDefine := isConstRcvr && rcvr.Val == "Data" && call.MethodName == "define"
		if !isStructNew && !isDataDefine {
			remaining = append(remaining, stmt)
			continue
		}
		// Found: Point = Struct.new(:x, :y) or Point = Data.define(:x, :y)
		className := constNode.Val
		// For scoped constant assignment (e.g., Diff::LCS::Change = Data.define(...)),
		// push intermediate modules so the class ends up in the right scope.
		var namespaceParts []string
		if constNode.Namespace != "" {
			namespaceParts = splitNamespace(constNode.Namespace)
			for _, part := range namespaceParts {
				r.PushModule(part, assign.LineNo())
			}
		}
		r.PushClass(className, assign.LineNo())
		cls := r.currentClass
		// Create attr_accessor for each symbol argument
		// (Data.define is immutable in Ruby, but we use attr_accessor for Go compatibility)
		for _, arg := range call.Args {
			if sym, ok := arg.(*SymbolNode); ok {
				name := sym.Val[1:] // strip leading ':'
				ivar := &IVar{Name: name, Readable: true, Writeable: true}
				cls.AddIVar(name, ivar)
			}
		}
		// Build field name set for ident→ivar rewriting
		fieldNames := make(map[string]bool)
		for _, arg := range call.Args {
			if sym, ok := arg.(*SymbolNode); ok {
				fieldNames[sym.Val[1:]] = true
			}
		}
		// Create initialize method with params for each field
		scope := NewScope("initialize")
		paramList := NewParamList()
		var bodyStmts Statements
		for _, arg := range call.Args {
			if sym, ok := arg.(*SymbolNode); ok {
				name := sym.Val[1:]
				paramList.AddParam(&Param{Name: name})
				bodyStmts = append(bodyStmts, &AssignmentNode{
					Left:  []Node{&IVarNode{Val: "@" + name, Class: cls}},
					Right: []Node{&IdentNode{Val: name}},
				})
			}
		}
		method := &Method{
			Name:      "initialize",
			Locals:    scope,
			Scope:     ScopeChain{scope},
			ParamList: paramList,
			Body: &Body{
				Statements: bodyStmts,
			},
		}
		cls.MethodSet.AddMethod(method)

		// Extract methods from block if present: Struct.new(:x, :y) do def ... end end
		if call.Block != nil {
			globalMS := r.MethodSetStack.Peek()
			for _, blockStmt := range call.Block.Body.Statements {
				if m, ok := blockStmt.(*Method); ok {
					// Remove from global MethodSet
					delete(globalMS.Methods, m.Name)
					newOrder := make([]string, 0, len(globalMS.Order))
					for _, name := range globalMS.Order {
						if name != m.Name {
							newOrder = append(newOrder, name)
						}
					}
					globalMS.Order = newOrder
					// Rewrite bare ident references to ivar references
					rewriteIdentsToIVars(m.Body.Statements, fieldNames, cls)
					// Add to class MethodSet
					cls.MethodSet.AddMethod(m)
				}
			}
		}

		r.PopClass()

		// For Data.define, add `with` method: returns a copy with specified fields overridden.
		// e.g., event.with(action: "-") → copy event, set Action = "-", return &copy
		if isDataDefine && cls.Type() != nil {
			classType := cls.Type().(*types.Class)
			fieldNames := []string{}
			for _, arg := range call.Args {
				if sym, ok := arg.(*SymbolNode); ok {
					fieldNames = append(fieldNames, sym.Val[1:])
				}
			}
			kwargsSpec := make([]types.KwargSpec, len(fieldNames))
			for i, name := range fieldNames {
				kwargsSpec[i] = types.KwargSpec{Name: name}
			}
			fields := fieldNames // capture for closure
			classType.Instance.Def("with", types.MethodSpec{
				ReturnType: func(receiverType types.Type, blockReturnType types.Type, args []types.Type) (types.Type, error) {
					return receiverType, nil
				},
				KwargsSpec: kwargsSpec,
				TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
					copyIdent := it.New("copy_")
					// copy_ := *receiver
					copyStmt := bst.Define(copyIdent, &ast.StarExpr{X: rcvr.Expr})
					stmts := []ast.Stmt{copyStmt}
					// For each provided kwarg, set the field on the copy
					for i, name := range fields {
						if i < len(args) && args[i].Expr != nil {
							stmts = append(stmts, bst.Assign(
								&ast.SelectorExpr{X: copyIdent, Sel: it.Get(GoName(name))},
								args[i].Expr,
							))
						}
					}
					return types.Transform{
						Stmts: stmts,
						Expr:  &ast.UnaryExpr{Op: token.AND, X: copyIdent},
					}
				},
			})
		}

		// Pop intermediate modules pushed for scoped constant assignment.
		// Use lightweight pop — modules already exist and have types from grammar
		// parsing. Full PopModule would re-create types and corrupt state.
		if len(namespaceParts) > 0 {
			for range namespaceParts {
				r.moduleStack.Pop()
				r.MethodSetStack.Pop()
				r.State.Pop()
				r.ScopeChain = r.ScopeChain[:len(r.ScopeChain)-1]
			}
		}
		// Don't add the assignment to remaining statements
	}
	r.Statements = remaining
}

// splitNamespace splits a "::" separated namespace (e.g., "Diff::LCS") into parts.
func splitNamespace(ns string) []string {
	return strings.Split(ns, "::")
}

// rewriteIdentsToIVars rewrites IdentNode references matching field names to IVarNode
// references in a Struct method body. This handles the Ruby pattern where Struct methods
// can reference fields as bare identifiers (which call the getter).
func rewriteIdentsToIVars(stmts Statements, fieldNames map[string]bool, cls *Class) {
	for i, stmt := range stmts {
		stmts[i] = rewriteNode(stmt, fieldNames, cls)
	}
}

func rewriteNode(n Node, fields map[string]bool, cls *Class) Node {
	switch node := n.(type) {
	case *IdentNode:
		if fields[node.Val] {
			return &IVarNode{Val: "@" + node.Val, Class: cls, Pos: Pos{lineNo: node.lineNo}}
		}
	case *StringNode:
		for idx, interps := range node.Interps {
			for j, interp := range interps {
				node.Interps[idx][j] = rewriteNode(interp, fields, cls)
			}
		}
	case *MethodCall:
		if node.Receiver != nil {
			node.Receiver = rewriteNode(node.Receiver, fields, cls)
		}
		for j, arg := range node.Args {
			node.Args[j] = rewriteNode(arg, fields, cls)
		}
	case *InfixExpressionNode:
		node.Left = rewriteNode(node.Left, fields, cls)
		node.Right = rewriteNode(node.Right, fields, cls)
	case *AssignmentNode:
		for j, right := range node.Right {
			node.Right[j] = rewriteNode(right, fields, cls)
		}
	case *ReturnNode:
		for j, val := range node.Val {
			node.Val[j] = rewriteNode(val, fields, cls)
		}
	case *Condition:
		node.Condition = rewriteNode(node.Condition, fields, cls)
		rewriteIdentsToIVars(node.True, fields, cls)
		if node.False != nil {
			if falseStmts, ok := node.False.(Statements); ok {
				rewriteIdentsToIVars(falseStmts, fields, cls)
			}
		}
	}
	return n
}

func (r *Root) Analyze() error {
	r.expandStructDefinitions()
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

	// Pre-pass: register module-level constants before the first pass so
	// that method bodies inside modules can resolve them.
	Tracer.SetPhase("module-constants")
	for _, mod := range r.TopLevelModules {
		Tracer.Record("analyze-module-constants", mod.name)
		r.analyzeModuleConstants(mod, r.ScopeChain)
	}

	// First pass, just to pick up method calls
	Tracer.SetPhase("first-pass (top-level statements)")
	if len(r.Statements) > 0 {
		err := (&Body{Statements: r.Statements}).InferReturnType(r.ScopeChain, nil)
		if err != nil {
			Tracer.Record("error", err.Error())
			if parseError, ok := err.(*ParseError); ok && parseError.terminal {
				return err
			}
		}
	}

	// Resolve scoped cross-module calls (e.g. Diff::LCS::Internals.lcs) that
	// couldn't be resolved during parsing because the target module wasn't loaded
	// yet. Must run before module-bodies so that param types are available when
	// method bodies are first analyzed.
	Tracer.SetPhase("deferred-scoped-calls")
	r.resolveDeferredScopedCalls()

	// Analyze module body statements (constants, etc.) and module classes (recursive).
	// Errors from gem-loaded modules are demoted to warnings so they don't block
	// analysis of user code.
	Tracer.SetPhase("module-bodies")
	for _, mod := range r.TopLevelModules {
		Tracer.Record("analyze-module", mod.name)
		if err := r.analyzeModule(mod, r.ScopeChain); err != nil {
			if mod.fromGem {
				fmt.Fprintf(os.Stderr, "warning: module %s: %v (continuing)\n", mod.name, err)
			} else {
				return err
			}
		}
	}

	// Pre-pass: analyze initialize methods in forward order so that ivar
	// types are available when child classes reference inherited fields.
	Tracer.SetPhase("initialize-pre-pass")
	r.preAnalyzeInitializers()

	// Work backwards through class declarations so that child classes are
	// analyzed before parents and method calls on inherited methods propagate
	// upward
	Tracer.SetPhase("class-method-sets (reverse order)")
	for i := len(r.Classes) - 1; i >= 0; i-- {
		class := r.Classes[i]
		Tracer.Record("analyze-class", class.name)
		if err := r.AnalyzeMethodSet(class.MethodSet, class.Type()); err != nil {
			return err
		}
	}
	Tracer.SetPhase("global-method-set")
	if err := r.AnalyzeMethodSet(r.MethodSetStack.Peek(), nil); err != nil {
		return err
	}

	// Eagerly analyze class method bodies that haven't been analyzed yet.
	// The ReturnType closure only fires when the method is called, but the
	// compiler emits all class methods. Run after all method sets are analyzed
	// so constants and instance types are fully resolved.
	Tracer.SetPhase("class-method-bodies")
	r.analyzeClassMethodBodies()

	// Before the second pass, clear cached types on statements that
	// contain unresolved inner calls (e.g., mixin methods whose block
	// params couldn't be typed in the first pass).
	Tracer.SetPhase("second-pass (re-infer top-level)")
	r.clearUnresolvedTypes(r.Statements)

	if len(r.Statements) > 0 {
		if err := (&Body{Statements: r.Statements}).InferReturnType(r.ScopeChain, nil); err != nil {
			// probably this is too aggressive
			return err
		}
	}

	Tracer.SetPhase("loose-call-resolution")
	for _, calls := range r.MethodSetStack.Peek().Calls {
		for _, c := range calls {
			GetType(c, r.ScopeChain, r.MethodSetStack.Peek().Class)
		}
	}
	return nil
}

// clearUnresolvedTypes walks statements and clears cached types on MethodCalls
// that wrap inner calls with nil types (e.g., mixin methods on user-defined
// types that couldn't be resolved in the first pass).
func (r *Root) clearUnresolvedTypes(stmts []Node) {
	for _, stmt := range stmts {
		r.clearUnresolvedNode(stmt)
	}
}

func (r *Root) clearUnresolvedNode(n Node) bool {
	switch node := n.(type) {
	case *MethodCall:
		childNeedsReval := false
		for _, arg := range node.Args {
			if r.clearUnresolvedNode(arg) {
				childNeedsReval = true
			}
		}
		if node.Receiver != nil {
			if r.clearUnresolvedNode(node.Receiver) {
				childNeedsReval = true
			}
		}
		// If this call itself has nil type, or a child was cleared, propagate
		if node.Type() == nil || childNeedsReval {
			node.SetType(nil)
			return true
		}
	}
	return false
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

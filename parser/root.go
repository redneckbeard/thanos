package parser

import (
	"fmt"
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
}

func NewRoot() *Root {
	globalMethodSet = NewMethodSet()
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
			// Skip methods already registered
			if modClass.HasMethod(m.Name) {
				continue
			}
			m.Scope = append(m.Scope[:len(m.Scope)-1], ScopeChain{module, m.Scope[len(m.Scope)-1]}...)
			cm := m
			funcName := pkgName + "." + GoName(m.Name)
			modClass.Def(m.Name, types.MethodSpec{
				ReturnType: func(receiverType types.Type, blockReturnType types.Type, args []types.Type) (types.Type, error) {
					if cm.Body.ReturnType == nil {
						for i, param := range cm.Params {
							if i < len(args) {
								param._type = args[i]
								cm.Locals.Set(param.Name, &RubyLocal{_type: args[i]})
							}
						}
						cm.Body.InferReturnType(cm.Scope, nil)
					}
					return cm.ReturnType(), nil
				},
				TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
					t := types.Transform{
						Expr: bst.Call(nil, funcName, types.UnwrapTypeExprs(args)...),
					}
					if modClass.PackagePath != "" {
						t.Imports = []string{modClass.PackagePath}
					}
					return t
				},
			})
		}
	} else if len(module.ClassMethods) > 0 {
		// Module didn't have a type yet — create one (same as PopModule)
		modClass := types.NewClass(module.name, "Object", nil, types.ClassRegistry)
		modClass.UserDefined = true
		modClass.Package = pkgName
		for _, m := range module.ClassMethods {
			m.Scope = append(m.Scope[:len(m.Scope)-1], ScopeChain{module, m.Scope[len(m.Scope)-1]}...)
			cm := m
			funcName := pkgName + "." + GoName(m.Name)
			modClass.Def(m.Name, types.MethodSpec{
				ReturnType: func(receiverType types.Type, blockReturnType types.Type, args []types.Type) (types.Type, error) {
					if cm.Body.ReturnType == nil {
						for i, param := range cm.Params {
							if i < len(args) {
								param._type = args[i]
								cm.Locals.Set(param.Name, &RubyLocal{_type: args[i]})
							}
						}
						cm.Body.InferReturnType(cm.Scope, nil)
					}
					return cm.ReturnType(), nil
				},
				TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
					t := types.Transform{
						Expr: bst.Call(nil, funcName, types.UnwrapTypeExprs(args)...),
					}
					if modClass.PackagePath != "" {
						t.Imports = []string{modClass.PackagePath}
					}
					return t
				},
			})
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
		r.MethodSetStack.Push(existing.MethodSet)
		r.moduleStack.Push(existing)
		r.ScopeChain = r.ScopeChain.Extend(existing)
		return
	}
	mod := &Module{name: name, Pos: Pos{lineNo: lineNo}}
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
			// Insert module scope before method's locals (last element)
			m.Scope = append(m.Scope[:len(m.Scope)-1], ScopeChain{module, m.Scope[len(m.Scope)-1]}...)
			cm := m
			funcName := pkgName + "." + GoName(m.Name)
			modClass.Def(m.Name, types.MethodSpec{
				ReturnType: func(receiverType types.Type, blockReturnType types.Type, args []types.Type) (types.Type, error) {
					if cm.Body.ReturnType == nil {
						for i, param := range cm.Params {
							if i < len(args) {
								param._type = args[i]
								cm.Locals.Set(param.Name, &RubyLocal{_type: args[i]})
							}
						}
						cm.Body.InferReturnType(cm.Scope, nil)
					}
					return cm.ReturnType(), nil
				},
				TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
					t := types.Transform{
						Expr: bst.Call(nil, funcName, types.UnwrapTypeExprs(args)...),
					}
					if modClass.PackagePath != "" {
						t.Imports = []string{modClass.PackagePath}
					}
					return t
				},
			})
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

func (r *Root) analyzeClassMethodBodies() {
	// Analyze class methods on top-level classes
	for _, cls := range r.Classes {
		for _, m := range cls.ClassMethods {
			if m.Body.ReturnType == nil {
				m.Body.InferReturnType(m.Scope, cls)
			}
		}
	}
	// Analyze class methods in module classes (recursive)
	for _, mod := range r.TopLevelModules {
		r.analyzeModuleClassMethods(mod)
	}
}

func (r *Root) analyzeModule(mod *Module, parentScope ScopeChain) error {
	modScope := parentScope.Extend(mod)
	if len(mod.Statements) > 0 {
		if _, err := GetType(mod.Statements, modScope, nil); err != nil {
			return err
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
	for _, cls := range mod.Classes {
		for _, m := range cls.ClassMethods {
			if m.Body.ReturnType == nil {
				m.Body.InferReturnType(m.Scope, cls)
			}
		}
	}
	for _, sub := range mod.Modules {
		r.analyzeModuleClassMethods(sub)
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

	// First pass, just to pick up method calls
	if len(r.Statements) > 0 {
		err := (&Body{Statements: r.Statements}).InferReturnType(r.ScopeChain, nil)
		if err != nil {
			if parseError, ok := err.(*ParseError); ok && parseError.terminal {
				return err
			}
		}
	}


	// Analyze module body statements (constants, etc.) and module classes (recursive)
	for _, mod := range r.TopLevelModules {
		if err := r.analyzeModule(mod, r.ScopeChain); err != nil {
			return err
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

	// Eagerly analyze class method bodies that haven't been analyzed yet.
	// The ReturnType closure only fires when the method is called, but the
	// compiler emits all class methods. Run after all method sets are analyzed
	// so constants and instance types are fully resolved.
	r.analyzeClassMethodBodies()

	// Before the second pass, clear cached types on statements that
	// contain unresolved inner calls (e.g., mixin methods whose block
	// params couldn't be typed in the first pass).
	r.clearUnresolvedTypes(r.Statements)

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

package parser

import "github.com/redneckbeard/thanos/types"

type Local interface {
	Type() types.Type
}

type local struct{}

func (loc *local) Type() types.Type {
	return nil
}

type RubyLocal struct {
	_type       types.Type
	Calls       []*MethodCall
	isRefinable bool // true if this variable's type can be refined (e.g., empty arrays)
}

func (rl *RubyLocal) String() string       { return rl._type.String() }
func (rl *RubyLocal) Type() types.Type     { return rl._type }
func (rl *RubyLocal) SetType(t types.Type) { rl._type = t }
func (rl *RubyLocal) AddCall(c *MethodCall) {
	rl.Calls = append(rl.Calls, c)
}

// MarkAsRefinable marks this variable as having a type that can be refined
func (rl *RubyLocal) MarkAsRefinable() {
	rl.isRefinable = true
}

// IsRefinable returns true if this variable's type can be refined
func (rl *RubyLocal) IsRefinable() bool {
	return rl.isRefinable
}

var BadLocal = new(local)

type Scope interface {
	Get(string) (Local, bool)
	Set(string, Local)
	Name() string
	TakesConstants() bool
}

type SimpleScope struct {
	name   string
	locals map[string]Local
}

func NewScope(name string) *SimpleScope {
	return &SimpleScope{name: name, locals: make(map[string]Local)}
}

func (scope *SimpleScope) Name() string {
	return scope.name
}

func (scope *SimpleScope) TakesConstants() bool {
	return false
}

func (scope *SimpleScope) Get(name string) (Local, bool) {
	if local, ok := scope.locals[name]; ok {
		return local, ok
	} else {
		return local, ok
	}
}

func (scope *SimpleScope) Set(name string, local Local) {
	scope.locals[name] = local
}

// Each iterates over all locals in this scope, calling fn for each.
func (scope *SimpleScope) Each(fn func(name string, local Local)) {
	for name, local := range scope.locals {
		fn(name, local)
	}
}

type ScopeChain []Scope

func NewScopeChain() ScopeChain {
	return ScopeChain{NewScope("")}
}

func (chain ScopeChain) Name() string {
	return chain[len(chain)-1].Name()
}

func (chain ScopeChain) Get(name string) (Local, bool) {
	return chain[len(chain)-1].Get(name)
}

func (chain ScopeChain) MustGet(name string) Local {
	local, got := chain[len(chain)-1].Get(name)
	if got {
		return local
	}
	panic("Called MustGet on Scope but no such local: " + name)
}

func (chain ScopeChain) Set(name string, local Local) {
	chain[len(chain)-1].Set(name, local)
}

// RefineVariableType updates a variable's type if it's marked as refinable
func (chain ScopeChain) RefineVariableType(name string, newType types.Type) bool {
	if local := chain.ResolveVar(name); local != BadLocal {
		if rubyLocal, ok := local.(*RubyLocal); ok && rubyLocal.IsRefinable() {
			rubyLocal.SetType(newType)
			return true
		}
	}
	return false
}

func (chain ScopeChain) ResolveVar(s string) Local {
	if len(chain) == 0 {
		return BadLocal
	}
	for i := len(chain) - 1; i >= 0; i-- {
		scope := chain[i]
		if t, found := scope.Get(s); found {
			return t
		}
	}
	return BadLocal
}

func (chain ScopeChain) Current() Scope {
	return chain[len(chain)-1]
}

func (chain ScopeChain) Extend(scope Scope) ScopeChain {
	dst := make(ScopeChain, len(chain))
	copy(dst, chain)
	return append(dst, scope)
}

func (chain ScopeChain) Prefix() string {
	var prefix string
	for _, scope := range chain {
		if _, ok := scope.(ConstantScope); ok {
			prefix += scope.Name()
		}
	}
	return prefix
}

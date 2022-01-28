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
	_type types.Type
	Calls []*MethodCall
}

func (rl *RubyLocal) String() string       { return rl._type.String() }
func (rl *RubyLocal) Type() types.Type     { return rl._type }
func (rl *RubyLocal) SetType(t types.Type) { rl._type = t }
func (rl *RubyLocal) AddCall(c *MethodCall) {
	rl.Calls = append(rl.Calls, c)
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

package parser

import (
	"github.com/redneckbeard/thanos/types"
)

type State string

type Local interface {
	Type() types.Type
}

const (
	TopLevelStatement  State = "TopLevelStatement"
	InClassBody        State = "InClassBody"
	InMethodDefinition State = "InMethodDefinition"
	InString           State = "InString"
)

type local struct{}

func (loc *local) Type() types.Type {
	return nil
}

var BadLocal = new(local)

type Scope interface {
	Get(string) (Local, bool)
	Set(string, Local)
	Name() string
	IsClass() bool
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

func (scope *SimpleScope) IsClass() bool {
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

func (chain ScopeChain) Extend(scope Scope) ScopeChain {
	dst := make(ScopeChain, len(chain))
	copy(dst, chain)
	return append(dst, scope)
}

type FSM struct {
	StateStack []State
	ScopeChain ScopeChain
}

func (fsm *FSM) PushState(s State) {
	fsm.StateStack = append(fsm.StateStack, s)
}

func (fsm *FSM) PopState() {
	if len(fsm.StateStack) > 0 {
		fsm.StateStack = fsm.StateStack[:len(fsm.StateStack)-1]
	}
}

func (fsm *FSM) CurrentState() State {
	if len(fsm.StateStack) == 0 {
		return TopLevelStatement
	}
	return fsm.StateStack[len(fsm.StateStack)-1]
}

func (fsm *FSM) PushScope(locals Scope) {
	fsm.ScopeChain = append(fsm.ScopeChain, locals)
}

func (fsm *FSM) PopScope() {
	if len(fsm.ScopeChain) > 0 {
		fsm.ScopeChain = fsm.ScopeChain[:len(fsm.ScopeChain)-1]
	}
}

package parser

type State string

const (
	TopLevelStatement  State = "TopLevelStatement"
	InClassBody        State = "InClassBody"
	InMethodDefinition State = "InMethodDefinition"
	InString           State = "InString"
)

type StateMachine struct {
	StateStack []State
	ScopeChain ScopeChain
}

func (fsm *StateMachine) PushState(s State) {
	fsm.StateStack = append(fsm.StateStack, s)
}

func (fsm *StateMachine) PopState() {
	if len(fsm.StateStack) > 0 {
		fsm.StateStack = fsm.StateStack[:len(fsm.StateStack)-1]
	}
}

func (fsm *StateMachine) CurrentState() State {
	if len(fsm.StateStack) == 0 {
		return TopLevelStatement
	}
	return fsm.StateStack[len(fsm.StateStack)-1]
}

func (fsm *StateMachine) PushScope(locals Scope) {
	fsm.ScopeChain = append(fsm.ScopeChain, locals)
}

func (fsm *StateMachine) PopScope() {
	if len(fsm.ScopeChain) > 0 {
		fsm.ScopeChain = fsm.ScopeChain[:len(fsm.ScopeChain)-1]
	}
}

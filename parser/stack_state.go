//compiled from https://github.com/whitequark/parser/blob/master/lib/parser/lexer/stack_state.rb
package parser

type StackState struct {
	name  string
	stack int
}

func NewStackState(name string) *StackState {
	return &StackState{name: name}
}

func (s *StackState) Clear() int {
	s.stack = 0
	return s.stack
}

func (s *StackState) Push(bit bool) bool {
	var bit_value int
	if bit {
		bit_value = 1
	} else {
		bit_value = 0
	}
	s.stack = s.stack<<1 | bit_value
	return bit
}

func (s *StackState) Pop() bool {
	bit_value := s.stack & 1
	s.stack >>= 1
	return bit_value == 1
}

func (s *StackState) Lexpop() bool {
	s.stack = s.stack>>1 | s.stack&1
	return s.stack&(1<<0)>>0 == 1
}

func (s *StackState) IsActive() bool {
	return s.stack&(1<<0)>>0 == 1
}

func (s *StackState) IsEmpty() bool {
	return s.stack == 0
}

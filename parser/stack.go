package parser

type Stack[T any] struct {
	stack []T
}

func (s *Stack[T]) Push(t T) {
	s.stack = append(s.stack, t)
}

func (s *Stack[T]) Pop() T {
	if len(s.stack) == 0 {
		var zero T
		return zero
	}
	last := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return last
}

func (s *Stack[T]) Peek() T {
	if len(s.stack) == 0 {
		var zero T
		return zero
	}
	return s.stack[len(s.stack)-1]
}

func (s *Stack[T]) Size() int {
	return len(s.stack)
}

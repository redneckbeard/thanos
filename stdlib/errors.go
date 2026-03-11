package stdlib

type RubyError struct {
	Msg string
}

func (e *RubyError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "RubyError"
}

type StandardError struct {
	RubyError
}

type RuntimeError struct {
	StandardError
}

type ArgumentError struct {
	StandardError
}

type TypeError struct {
	StandardError
}

type ZeroDivisionError struct {
	StandardError
}

type NameError struct {
	StandardError
}

type IndexError struct {
	StandardError
}

type KeyError struct {
	StandardError
}

type RangeError struct {
	StandardError
}

type IOError struct {
	StandardError
}

type NotImplementedError struct {
	StandardError
}

type StopIteration struct {
	StandardError
}

type RegexpError struct {
	StandardError
}

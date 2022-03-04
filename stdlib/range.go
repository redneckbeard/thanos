package stdlib

type Range[T comparable] struct {
	Lower, Upper T
	Inclusive    bool
}

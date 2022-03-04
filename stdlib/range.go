package stdlib

type Rangeable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64 | ~string
}

type Range[T Rangeable] struct {
	Lower, Upper T
	Inclusive    bool
}

func RangeCovers[T Rangeable](r *Range[T], t T) bool {
	return t >= r.Lower && t <= r.Upper
}

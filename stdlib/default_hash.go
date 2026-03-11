package stdlib

// DefaultHash wraps an OrderedMap with a default value function.
// When a key is missing, Get calls the factory function which may set the key.
type DefaultHash[K comparable, V any] struct {
	*OrderedMap[K, V]
	DefaultFn func(*OrderedMap[K, V], K) V
}

func NewDefaultHash[K comparable, V any](factory func(*OrderedMap[K, V], K) V) *DefaultHash[K, V] {
	return &DefaultHash[K, V]{
		OrderedMap: NewOrderedMap[K, V](),
		DefaultFn:  factory,
	}
}

func NewDefaultHashWithValue[K comparable, V any](val V) *DefaultHash[K, V] {
	return &DefaultHash[K, V]{
		OrderedMap: NewOrderedMap[K, V](),
		DefaultFn: func(m *OrderedMap[K, V], k K) V {
			m.Set(k, val)
			return val
		},
	}
}

func (h *DefaultHash[K, V]) Get(key K) V {
	if v, ok := h.Data[key]; ok {
		return v
	}
	return h.DefaultFn(h.OrderedMap, key)
}

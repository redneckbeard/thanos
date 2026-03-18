package stdlib

func Compact[T any](arr []*T) []T {
	result := make([]T, 0, len(arr))
	for _, v := range arr {
		if v != nil {
			result = append(result, *v)
		}
	}
	return result
}

// Ptr returns a pointer to the given value, enabling &literal syntax in Go.
func Ptr[T any](v T) *T {
	return &v
}

func OrDefault[T any](val *T, def T) T {
	if val != nil {
		return *val
	}
	return def
}

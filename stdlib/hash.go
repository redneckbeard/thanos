package stdlib

import "reflect"

func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := []K{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func MapValues[K comparable, V any](m map[K]V) []V {
	vals := []V{}
	for _, v := range m {
		vals = append(vals, v)
	}
	return vals
}

func MapMerge[K comparable, V any](base, other map[K]V) map[K]V {
	merged := map[K]V{}
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range other {
		merged[k] = v
	}
	return merged
}

func MapHasValue[K comparable, V any](m map[K]V, val V) bool {
	for _, v := range m {
		if reflect.DeepEqual(v, val) {
			return true
		}
	}
	return false
}

func MapInvert[K comparable, V comparable](m map[K]V) map[V]K {
	inverted := map[V]K{}
	for k, v := range m {
		inverted[v] = k
	}
	return inverted
}


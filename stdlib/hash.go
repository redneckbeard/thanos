package stdlib

import "reflect"

func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := []K{}
	for k, _ := range m {
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

func MapHasValue[K comparable, V any](m map[K]V, val V) bool {
	for _, v := range m {
		if reflect.DeepEqual(v, val) {
			return true
		}
	}
	return false
}

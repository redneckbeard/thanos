package stdlib

import "reflect"

func SubtractSlice[T any](left, right []T) []T {
	for _, x := range right {
		indices := []int{}
		for i, y := range left {
			if reflect.DeepEqual(x, y) {
				indices = append([]int{i}, indices...)
			}
		}
		for _, i := range indices {
			left = append(left[:i], left[i+1:]...)
		}
	}
	return left
}

func Uniq[T comparable](arr []T) []T {
	set := make(map[T]bool)
	order := []T{}
	for _, elem := range arr {
		if _, ok := set[elem]; !ok {
			set[elem] = true
			order = append(order, elem)
		}
	}
	return order
}

package stdlib

import (
	"fmt"
	"strings"
)

type Set[T comparable] map[T]bool

func NewSet[T comparable](elems []T) Set[T] {
	set := make(Set[T])
	for _, elem := range elems {
		set[elem] = true
	}
	return set
}

func (s Set[T]) String() string {
	var elems []string
	for elem := range s {
		elems = append(elems, fmt.Sprintf("%v", elem))
	}
	return fmt.Sprintf("Set{%s}", strings.Join(elems, ", "))
}

func (s Set[T]) Union(s2 Set[T]) Set[T] {
	set := make(Set[T])
	for k := range s {
		set[k] = true
	}
	for k := range s2 {
		set[k] = true
	}
	return set
}

func (s Set[T]) Difference(s2 Set[T]) Set[T] {
	set := make(Set[T])
	for k := range s {
		if _, ok := s2[k]; !ok {
			set[k] = true
		}
	}
	return set
}

func (s Set[T]) Intersection(s2 Set[T]) Set[T] {
	set := make(Set[T])
	for k := range s {
		if _, ok := s2[k]; ok {
			set[k] = true
		}
	}
	return set
}

func (s Set[T]) Disjoint(s2 Set[T]) Set[T] {
	set := make(Set[T])
	for k := range s {
		if _, ok := s2[k]; !ok {
			set[k] = true
		}
	}
	for k := range s2 {
		if _, ok := s[k]; !ok {
			set[k] = true
		}
	}
	return set
}

func (s Set[T]) SupersetQ(s2 Set[T]) bool {
	if len(s2) > len(s) {
		return false
	}
	for k := range s2 {
		if _, ok := s[k]; !ok {
			return false
		}
	}
	return true
}

func (s Set[T]) ProperSupersetQ(s2 Set[T]) bool {
	if len(s2) >= len(s) {
		return false
	}
	for k := range s2 {
		if _, ok := s[k]; !ok {
			return false
		}
	}
	return true
}

func (s Set[T]) SubsetQ(s2 Set[T]) bool {
	if len(s2) < len(s) {
		return false
	}
	for k := range s {
		if _, ok := s2[k]; !ok {
			return false
		}
	}
	return true
}

func (s Set[T]) ProperSubsetQ(s2 Set[T]) bool {
	if len(s2) <= len(s) {
		return false
	}
	for k := range s {
		if _, ok := s2[k]; !ok {
			return false
		}
	}
	return true
}

func (s Set[T]) ToA() []T {
	var arr []T
	for k := range s {
		arr = append(arr, k)
	}
	return arr
}

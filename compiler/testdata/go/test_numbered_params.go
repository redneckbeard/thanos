package main

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
)

func main() {
	arr := []int{1, 2, 3}
	for _, _1 := range arr {
		fmt.Println(_1)
	}
	mapped := []int{}
	for _, _1 := range arr {
		mapped = append(mapped, _1*2)
	}
	result := mapped
	om := stdlib.NewOrderedMap[string, int]()
	om.Set("a", 1)
	om.Set("b", 2)
	h := om
	for _1, _2 := range h.All() {
		fmt.Printf("%s: %d\n", _1, _2)
	}
}

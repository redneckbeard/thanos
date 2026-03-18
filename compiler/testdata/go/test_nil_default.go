package main

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
)

func Greet(name *string) {
	if name == nil {
		v := "world"
		name = &v
	}
	fmt.Printf("hello %s\n", *name)
}
func Find_index(arr []int, target int) *int {
	for i, val := range arr {
		if val == target {
			return stdlib.Ptr[int](i)
		}
	}
	return nil
}
func main() {
	Greet(stdlib.Ptr[string]("paul"))
	Greet(nil)
	fmt.Println(*Find_index([]int{10, 20, 30}, 20))
	fmt.Println(Find_index([]int{10, 20, 30}, 99) == nil)
}

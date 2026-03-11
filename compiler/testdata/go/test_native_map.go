package main

import "fmt"

func main() {
	simple := map[string]int{"a": 1, "b": 2}
	fmt.Println(simple["a"])
	fmt.Println(len(simple))
	fmt.Println(len(simple) == 0)
	_, ok := simple["a"]
	fmt.Println(ok)
	simple["c"] = 3
}

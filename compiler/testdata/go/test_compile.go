package main

import "fmt"

func Foo(a, b int) []int {
	return []int{a, b}
}
func main() {
	fmt.Println(len(Foo(5, 7)))
}

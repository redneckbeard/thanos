package main

import "fmt"

func foo(a, b int) []int {
	return []int{a, b}
}
func main() {
	fmt.Println(len(foo(5, 7)))
}

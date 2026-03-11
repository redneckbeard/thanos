package main

import "fmt"

func main() {
	double := func(x int) int {
		return x * 2
	}
	fmt.Println(double(5))
	no_args := func() string {
		return "hello"
	}
	fmt.Println(no_args())
	add := func(a, b int) int {
		return a + b
	}
	fmt.Println(add(3, 4))
}

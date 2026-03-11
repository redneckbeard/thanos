package main

import "fmt"

func main() {
	// Top-level comment before code
	x := 42
	// Comment before conditional
	if x > 10 {
		fmt.Println("big")
	} else {
		fmt.Println("small")
	}
	// Comment before loop
	arr := []int{1, 2, 3}
	for _, n := range arr {
		fmt.Println(n)
	}
	// Comment before assignment
	y := x + 1
	// Comment before last statement
	fmt.Println(y)
	// Multiple consecutive comments
	// describe what the next
	// block of code does
	z := y * 2
	fmt.Println(z)
	// Block comments are also preserved
	// They span multiple lines
	a := z + 1
	fmt.Println(a)
}

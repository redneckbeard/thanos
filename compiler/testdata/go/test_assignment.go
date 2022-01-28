package main

import "fmt"

func main() {
	a, b := 1, 2.0
	c, d := a+1, b+2
	e := []int{1, 2, 3}
	m, n := e[0], e[1]
	fmt.Println(float64(c) + d)
}

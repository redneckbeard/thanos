package main

import "fmt"

func main() {
	x := 100
	for x > 0 {
		x--
	}
	for x != 50 {
		x++
	}
	y := 0
	for x {
		y++
		if y > 5 {
			break
		}
	}
	for !(y > 100) {
		y++
		if y%2 == 0 {
			continue
		}
		fmt.Println(y)
	}
}

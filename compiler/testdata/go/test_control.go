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
	var x int
	for _, x = range []int{1, 2, 3, 4} {
		fmt.Println(x)
		if x == 3 {
			break
		}
	}
	var k string
	var v int
	for k, v = range map[string]int{"foo": 1, "bar": 2, "baz": 3, "quux": 4} {
		if k == "foo" || v == 10 {
			continue
		}
	}
}

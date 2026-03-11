package main

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
)

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
	om := stdlib.NewOrderedMap[string, int]()
	om.Set("foo", 1)
	om.Set("bar", 2)
	om.Set("baz", 3)
	om.Set("quux", 4)
	for k, v = range om.All() {
		if k == "foo" || v == 10 {
			continue
		}
	}
}

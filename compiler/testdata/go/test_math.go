package main

import (
	"fmt"
	"math"
)

func main() {
	x := 10 / 2
	y := float64(x) / 2.0
	z := int(math.Pow(float64(x), 2))
	a := int(math.Pow(float64(x), float64(x)))
	b := math.Pow(y, 2)
	c := 12.0 / 4
	d := int(math.Abs(-50))
	e := int(math.Abs(float64(x)))
	for x := 0; x < 10; x++ {
		if x%2 == 0 {
			fmt.Println(x)
		}
	}
	for x := 15; x >= 10; x-- {
		if x%2 == 1 {
			fmt.Println(x)
		}
	}
	for x := -5; x <= 5; x++ {
		switch {
		case x == 0:
			fmt.Println("zero")
		case x > 0:
			fmt.Println("positive")
		case x < 0:
			fmt.Println("negative")
		}
	}
}

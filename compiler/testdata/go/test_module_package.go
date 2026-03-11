package main

import (
	"fmt"
	"tmpmod/geometry"

	"github.com/redneckbeard/thanos/stdlib"
)

func main() {
	c := geometry.NewCircle(10)
	fmt.Println(stdlib.FormatFloat(c.Area()))
}

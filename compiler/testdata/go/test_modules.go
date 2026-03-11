package main

import (
	"fmt"
	"tmpmod/foo"
)

const Quux = false

func main() {
	fmt.Println(foo.NewBaz().Quux())
}

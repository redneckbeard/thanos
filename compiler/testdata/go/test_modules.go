package main

import "fmt"

const FooQuux = "quux"
const FooBazQuux = 10
const Quux = false

type FooBaz struct {
}

func NewFooBaz() *FooBaz {
	newInstance := &FooBaz{}
	return newInstance
}
func (b *FooBaz) Quux() int {
	return FooBazQuux
}
func main() {
	fmt.Println(NewFooBaz().Quux())
}

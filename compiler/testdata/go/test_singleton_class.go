package main

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
)

type Greeter struct {
}

func NewGreeter() *Greeter {
	newInstance := &Greeter{}
	return newInstance
}

var GreeterClass = stdlib.NewMetaclass[Greeter]("Greeter")

func GreeterHello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
func main() {
	fmt.Println(GreeterHello("world"))
}

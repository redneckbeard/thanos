package main

import "fmt"

type Greeter struct {
}

func NewGreeter() *Greeter {
	newInstance := &Greeter{}
	return newInstance
}
func GreeterHello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
func main() {
	fmt.Println(GreeterHello("world"))
}

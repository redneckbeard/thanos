package main

import "fmt"

type Calculator struct {
}

func NewCalculator() *Calculator {
	newInstance := &Calculator{}
	return newInstance
}
func (c *Calculator) Double(x int) int {
	return x * 2
}
func (c *Calculator) Triple(x int) int {
	return x * 3
}

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
	c := NewCalculator(0)
	fmt.Println(c.Double(5))
	fmt.Println(c.Triple(4))
	fmt.Println(GreeterHello("world"))
}

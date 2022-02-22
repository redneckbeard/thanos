package main

import "fmt"

type Vehicle struct {
	Starting_miles int
	no_reader      string
}

func NewVehicle(starting_miles int) *Vehicle {
	return &Vehicle{Starting_miles: starting_miles, no_reader: "unexported"}
}
func (v *Vehicle) Drive(x int) int {
	v.Starting_miles += x
	return v.Starting_miles
}
func (v *Vehicle) Mileage() string {
	v.log()
	return fmt.Sprintf("%d miles", v.Starting_miles)
}
func (v *Vehicle) log() {
	fmt.Println("log was called")
}

type Car struct {
	Starting_miles int
	no_reader      string
}

func NewCar(starting_miles int) *Car {
	return &Car{Starting_miles: starting_miles, no_reader: "unexported"}
}
func (c *Car) Log() {
	fmt.Println("it's a different method!")
}
func (c *Car) Drive(x int) int {
	c.Starting_miles += x
	return c.Starting_miles
}
func (c *Car) Mileage() string {
	c.Log()
	return fmt.Sprintf("%d miles", c.Starting_miles)
}
func main() {
	mapped := []string{}
	for _, car := range []*Car{NewCar(10), NewCar(20), NewCar(30)} {
		car.Drive(100)
		mapped = append(mapped, fmt.Sprintf("%s, started at %d", car.Mileage(), car.Starting_miles))
	}
	fmt.Println(mapped)
}

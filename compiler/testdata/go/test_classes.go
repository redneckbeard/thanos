package main

import "fmt"

type Vehicle struct {
	Starting_miles int
	no_reader      string
}

func NewVehicle(starting_miles int) *Vehicle {
	return &Vehicle{Starting_miles: starting_miles, no_reader: "unexported"}
}
func (v *Vehicle) drive(x int) int {
	v.starting_miles += x
	return v.starting_miles
}
func (v *Vehicle) log() {
	fmt.Println("log was called")
}
func (v *Vehicle) mileage() string {
	v.log()
	return fmt.Sprintf("%d miles", v.starting_miles)
}

type Car struct {
	Starting_miles int
	no_reader      string
}

func NewCar(starting_miles int) *Car {
	return &Car{Starting_miles: starting_miles, no_reader: "unexported"}
}
func (c *Car) log() {
	fmt.Println("it's a different method!")
}
func (c *Car) drive(x int) int {
	c.starting_miles += x
	return c.starting_miles
}
func (c *Car) mileage() string {
	c.log()
	return fmt.Sprintf("%d miles", c.starting_miles)
}
func main() {
	mapped := []string{}
	for _, car := range []*Car{NewCar(10), NewCar(20), NewCar(30)} {
		car.drive(100)
		mapped = append(mapped, fmt.Sprintf("%s, started at %d", car.mileage(), car.Starting_miles))
	}
	cars := mapped
	fmt.Println(cars)
}

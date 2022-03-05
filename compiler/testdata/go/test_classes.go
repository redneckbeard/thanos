package main

import "fmt"

type Vehicle struct {
	starting_miles int
	no_reader      string
	registration   string
}

func NewVehicle(starting_miles int) *Vehicle {
	return &Vehicle{starting_miles: starting_miles, no_reader: "unexported"}
}
func (v *Vehicle) Drive(x int) int {
	v.starting_miles += x
	return v.starting_miles
}
func (v *Vehicle) Mileage() string {
	v.log()
	return fmt.Sprintf("%d miles", v.starting_miles)
}
func (v *Vehicle) log() {
	fmt.Println("log was called")
}
func (v *Vehicle) Starting_miles() int {
	return v.starting_miles
}
func (v *Vehicle) SetRegistration(registration string) string {
	v.registration = registration
	return v.registration
}

type Car struct {
	starting_miles int
	no_reader      string
	registration   string
}

func NewCar(starting_miles int) *Car {
	return &Car{starting_miles: starting_miles, no_reader: "unexported"}
}
func (c *Car) Log() {
	fmt.Println("it's a different method!")
}
func (c *Car) Drive(x int) int {
	c.starting_miles += x
	return c.starting_miles
}
func (c *Car) Mileage() string {
	c.log()
	return fmt.Sprintf("%d miles", c.starting_miles)
}
func (c *Car) Starting_miles() int {
	return c.starting_miles
}
func (c *Car) SetRegistration(registration string) string {
	c.registration = registration
	return c.registration
}
func main() {
	mapped := []string{}
	for _, car := range []*Car{NewCar(10), NewCar(20), NewCar(30)} {
		car.Drive(100)
		car.SetRegistration("XXXXXX")
		mapped = append(mapped, fmt.Sprintf("%s, started at %d", car.Mileage(), car.Starting_miles()))
	}
	fmt.Println(mapped)
}

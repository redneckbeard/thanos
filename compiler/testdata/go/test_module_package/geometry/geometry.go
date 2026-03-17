package geometry

import "github.com/redneckbeard/thanos/stdlib"

type Circle struct {
	radius int
}

func NewCircle(radius int) *Circle {
	newInstance := &Circle{}
	newInstance.Initialize(radius)
	return newInstance
}

var CircleClass = stdlib.NewMetaclass[Circle]("GeometryCircle")

func (c *Circle) Initialize(radius int) int {
	c.radius = radius
	return c.radius
}
func (c *Circle) Area() float64 {
	return Pi() * float64(c.radius) * float64(c.radius)
}
func Pi() float64 {
	return 3.14
}

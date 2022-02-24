package main

import "fmt"

const CTDriverLICENSE_AGE = 18
const CrossStateCommercialCTDriverLICENSE_AGE = 21
const PERMIT_AGE = 16

type CTDriver struct {
	age int
}

func NewCTDriver(age int) *CTDriver {
	return &CTDriver{age: age}
}
func (c *CTDriver) Can_drive() bool {
	return c.age >= CTDriverLICENSE_AGE
}

type CrossStateCommercialCTDriver struct {
	age int
}

func NewCrossStateCommercialCTDriver(age int) *CrossStateCommercialCTDriver {
	return &CrossStateCommercialCTDriver{age: age}
}
func (c *CrossStateCommercialCTDriver) Can_drive() bool {
	return c.age >= CTDriverLICENSE_AGE
}
func main() {
	if NewCTDriver(19).Can_drive() {
		fmt.Println(CrossStateCommercialCTDriverLICENSE_AGE)
	}
}

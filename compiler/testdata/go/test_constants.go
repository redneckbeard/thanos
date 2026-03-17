package main

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
)

const CTDriverLICENSE_AGE = 18
const CTDriverKIND_MOTORCYCLE = "motorcycle"
const CTDriverKIND_COMMERCIAL = "cdl"
const CTDriverKIND_SCOOTER = "scooter"
const CrossStateCommercialCTDriverLICENSE_AGE = 21
const PERMIT_AGE = 16

var CTDriverLICENSE_KINDS []string = []string{CTDriverKIND_MOTORCYCLE, CTDriverKIND_COMMERCIAL, CTDriverKIND_SCOOTER}

type CTDriver struct {
	age int
}

func NewCTDriver(age int) *CTDriver {
	newInstance := &CTDriver{}
	newInstance.Initialize(age)
	return newInstance
}

var CTDriverClass = stdlib.NewMetaclass[CTDriver]("CTDriver")

func (c *CTDriver) Initialize(age int) int {
	c.age = age
	return c.age
}
func (c *CTDriver) Can_drive() bool {
	return c.age >= CTDriverLICENSE_AGE
}

type CrossStateCommercialCTDriver struct {
	age int
}

func NewCrossStateCommercialCTDriver() *CrossStateCommercialCTDriver {
	newInstance := &CrossStateCommercialCTDriver{}
	return newInstance
}

var CrossStateCommercialCTDriverClass = stdlib.NewMetaclass[CrossStateCommercialCTDriver]("CrossStateCommercialCTDriver")

func main() {
	if NewCTDriver(19).Can_drive() {
		fmt.Println(CrossStateCommercialCTDriverLICENSE_AGE)
	}
}

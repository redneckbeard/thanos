package main

type parentClass struct {
	shared_things []string
}

func (p *parentClass) Family_things() []string {
	// @@family_things
}
func (p *parentClass) Shared_things() []string {
	return p.shared_things
}

var ParentClass = &parentClass{}

type Parent struct {
	Class    *parentClass
	MyThings []string
}

func NewParent() *Parent {
	return &Parent{}
}

func (p *Parent) Family_things() []string {
	// self.class.family_things
}
func (p *Parent) Shared_things() []string {
	return p.Class.Shared_things()
}

type Child struct {
	MyThings []string
}

func NewChild() *Child {
	return &Child{}
}

func (p *Child) Family_things() []string {
	// self.class.family_things
}
func (p *Child) Shared_things() []string {
	// self.class.shared_things
}

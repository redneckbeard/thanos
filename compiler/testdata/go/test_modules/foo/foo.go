package foo

import "github.com/redneckbeard/thanos/stdlib"

const Quux = "quux"
const BazQuux = 10

type Baz struct {
}

func NewBaz() *Baz {
	newInstance := &Baz{}
	return newInstance
}

var BazClass = stdlib.NewMetaclass[Baz]("FooBaz")

func (b *Baz) Quux() int {
	return BazQuux
}

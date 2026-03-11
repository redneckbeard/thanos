package foo

const Quux = "quux"
const BazQuux = 10

type Baz struct {
}

func NewBaz() *Baz {
	newInstance := &Baz{}
	return newInstance
}
func (b *Baz) Quux() int {
	return BazQuux
}

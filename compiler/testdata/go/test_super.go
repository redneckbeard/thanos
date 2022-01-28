package main

import (
	"fmt"
	"math"
)

type Foo struct {
	baz float64
}

func NewFoo() *Foo {
	newInstance := &Foo{}
	return newInstance
}
func (f *Foo) A() bool {
	return true
}
func (f *Foo) B(x float64, y int) float64 {
	f.baz = x + float64(y)
	return f.baz
}

type Bar struct {
	baz float64
}

func NewBar() *Bar {
	newInstance := &Bar{}
	return newInstance
}
func (b *Bar) A() bool {
	super := func(b *Bar) bool {
		return true
	}
	return super(b)
}
func (b *Bar) B(x float64, y int) float64 {
	super := func(b *Bar, x float64, y int) float64 {
		b.baz = x + float64(y)
		return b.baz
	}
	return super(b, x, y)
}

type Baz struct {
	baz float64
}

func NewBaz() *Baz {
	newInstance := &Baz{}
	return newInstance
}
func (b *Baz) B(x float64, y int) float64 {
	super := func(b *Baz, x float64, y int) float64 {
		b.baz = x + float64(y)
		return b.baz
	}
	return super(b, 1.4, 2) + 1.0
}
func (b *Baz) A() bool {
	return true
}

type Quux struct {
	baz float64
}

func NewQuux() *Quux {
	newInstance := &Quux{}
	return newInstance
}
func (q *Quux) A() bool {
	super := func(q *Quux) bool {
		return true
	}
	anc := super(q)
	return !anc
}
func (q *Quux) B(x float64, y int) float64 {
	super := func(q *Quux, x float64, y int) float64 {
		q.baz = x + float64(y)
		return q.baz
	}
	anc := super(q, math.Pow(x, 2), int(math.Pow(float64(y), 2)))
	return anc + 1
}
func main() {
	NewBar().B(1.5, 2)
	NewBaz().B(1.5, 2)
	quux := NewQuux()
	if quux.A() {
		fmt.Println(quux.B(2.0, 4))
	}
}

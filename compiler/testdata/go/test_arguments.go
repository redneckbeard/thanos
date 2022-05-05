package main

import "fmt"

func Pos_and_kw(foo string, bar bool) {
	if bar {
		fmt.Println(foo)
	}
}
func All_kw(foo string, bar bool) {
	if bar {
		fmt.Println(foo)
	}
}
func Defaults(foo, bar string) string {
	return fmt.Sprintf("foo: %s, bar: %s", foo, bar)
}
func Splat(a int, c bool, b ...int) int {
	if c {
		return b[0]
	} else {
		return a
	}
}

func Double_splat(foo int, bar map[string]int) int {
	return foo + bar["baz"]
}

type Foo struct {
	foo int
}

func NewFoo(foo int) *Foo {
	newInstance := &Foo{}
	newInstance.Initialize(foo)
	return newInstance
}
func (f *Foo) Initialize(foo int) int {
	f.foo = foo
	return f.foo
}
func main() {
	Pos_and_kw("x", true)
	Pos_and_kw("x", false)
	All_kw("y", false)
	All_kw("z", false)
	Defaults("x", "y")
	Defaults("z", "y")
	Defaults("z", "a")
	NewFoo(10)
	Splat(9, false, 2, 3)
	Splat(9, true, 2)
	Splat(9, false)
	Splat(9, false, []int{1, 2}...)
	Splat(9, false, append([]int{5}, []int{1, 2}...)...)
	Double_splat(1, map[string]int{"bar": 2, "baz": 3})
	Double_splat(1, map[string]int{"baz": 3})
	Double_splat(1, map[string]int{"baz": 4})
	hash_from_elsewhere := map[string]int{"foo": 1, "baz": 4}
	hash_from_elsewhere_kwargs := map[string]int{}
	for k, v := range hash_from_elsewhere {
		switch k {
		case "foo":
		default:
			hash_from_elsewhere_kwargs[k] = v
		}
	}
	Double_splat(hash_from_elsewhere["foo"], hash_from_elsewhere_kwargs)
	foo := []int{1, 2, 3}
	a, b := foo[0], foo[1:len(foo)]
	c, d, e := foo[0], foo[1], foo[2:len(foo)]
	syms := []string{"foo", "bar", "baz"}
	f := append([]string{"quux"}, syms...)
	g, h, i := "quux", syms[0], syms[1]
	x, y, z := "quux", syms[0], syms[1:len(syms)]
}

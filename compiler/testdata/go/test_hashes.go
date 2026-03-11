package main

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
)

func main() {
	om := stdlib.NewOrderedMap[string, string]()
	om.Set("foo", "x")
	om.Set("bar", "y")
	h := om
	var val string
	if v, ok := h.Data["foo"]; ok {
		val = v
		h.Delete("foo")
	}
	x := val
	var val1 string
	if v, ok := h.Data["baz"]; ok {
		val1 = v
		h.Delete("baz")
	} else {
		k := "baz"
		val1 = fmt.Sprintf("default for %v", k)
	}
	y := val1
}

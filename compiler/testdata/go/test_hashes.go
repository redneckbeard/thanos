package main

import "fmt"

func main() {
	h := map[string]string{"foo": "x", "bar": "y"}
	var val string
	if v, ok := h["foo"]; ok {
		val = v
		delete(h, "foo")
	}
	x := val
	var val1 string
	if v, ok := h["baz"]; ok {
		val1 = v
		delete(h, "baz")
	} else {
		k := "baz"
		val1 = fmt.Sprintf("default for %v", k)
	}
	y := val1
}

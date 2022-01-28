package main

import "fmt"

func pos_and_kw(foo string, bar bool) {
	if bar {
		fmt.Println(foo)
	}
}
func all_kw(foo string, bar bool) {
	if bar {
		fmt.Println(foo)
	}
}
func defaults(foo, bar string) string {
	return fmt.Sprintf("foo: %s, bar: %s", foo, bar)
}
func main() {
	pos_and_kw("x", true)
	pos_and_kw("x", false)
	all_kw("y", false)
	all_kw("z", false)
	defaults("x", "y")
	defaults("z", "y")
	defaults("z", "a")
}

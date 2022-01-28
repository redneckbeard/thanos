package main

import "fmt"

func cond_return(a, b int) int {
	if a < 0 && b < 0 {
		return 0
	} else {
		if a >= b {
			return a
		} else {
			return b
		}
	}
}
func cond_assignment(a, b int, c bool) bool {
	var foo bool
	if a == b {
		foo = true
	} else {
		foo = false
	}
	return foo || c
}
func cond_invoke() int {
	fmt.Println("it's true")
	return 10
}
func tern(x, y, z int) int {
	if x == 10 {
		return y
	} else {
		return z
	}
}
func length_if_array(arr []string) int {
	return len(arr)
}
func puts_if_not_symbol() {
	fmt.Println("isn't a symbol")
}
func switch_on_int_val(x int) string {
	switch x {
	case 0:
		return "none"
	case 1:
		return "one"
	case 2, 3, 4, 5:
		return "a few"
	default:
		return "many"
	}
}
func switch_on_int_with_range(x int) string {
	switch {
	case x == 0:
		return "none"
	case x == 1:
		return "one"
	case x >= 2 && x <= 5:
		return "a few"
	case x == 6 || x == 7 || x == 8:
		return "several"
	default:
		return "many"
	}
}
func main() {
	baz := cond_return(2, 4)
	quux := cond_assignment(1, 3, false)
	zoo := cond_invoke()
	last := tern(10, 20, 30)
	length_if_array([]string{"foo", "bar", "baz"})
	switch_on_int_val(5)
	switch_on_int_with_range(5)
}

package main

import "fmt"

func Cond_return(a, b int) int {
	if a == 47 {
		return a * b
	}
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
func Cond_assignment(a, b int, c bool) bool {
	var foo bool
	if a == b {
		foo = true
	} else {
		foo = false
	}
	return foo || c
}
func Cond_invoke() int {
	fmt.Println("it's true")
	return 10
}
func Tern(x, y, z int) int {
	if !(z < 50) {
		return 99
	}
	if x == 10 {
		return y
	} else {
		return z
	}
}
func Length_if_array(arr []string) int {
	return len(arr)
}
func Puts_if_not_symbol() {
	fmt.Println("isn't a symbol")
}
func Switch_on_int_val(x int) string {
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
func Switch_on_int_with_range(x int) string {
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
	baz := Cond_return(2, 4)
	quux := Cond_assignment(1, 3, false)
	zoo := Cond_invoke()
	last := Tern(10, 20, 30)
	Length_if_array([]string{"foo", "bar", "baz"})
	Switch_on_int_val(5)
	Switch_on_int_with_range(5)
}

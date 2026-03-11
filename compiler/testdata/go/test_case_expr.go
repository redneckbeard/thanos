package main

import "fmt"

func Describe(n int) {
	var result string
	switch n {
	case 1:
		result = "one"
	case 2:
		result = "two"
	default:
		result = "other"
	}
	fmt.Println(result)
}
func Categorize(n int) string {
	switch n {
	case 1:
		return "small"
	case 2:
		return "medium"
	default:
		return "large"
	}
}
func main() {
	Describe(1)
	Categorize(1)
}

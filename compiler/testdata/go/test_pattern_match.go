package main

import "fmt"

func main() {
	arr := []int{1, 2, 3}
	if len(arr) == 3 {
		a := arr[0]
		b := arr[1]
		c := arr[2]
		fmt.Println(a)
		fmt.Println(b)
		fmt.Println(c)
	} else if len(arr) == 2 {
		a := arr[0]
		fmt.Println(a)
	}
}

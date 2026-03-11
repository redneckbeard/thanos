package main

func Make_arr(a, b, c int) []int {
	arr := []int{a, b, c}
	arr = append(arr, a*b*c)
	if a > 10 {
		return []int{b}
	}
	return arr
}
func Sum(a []int) int {
	acc := 0
	for _, n := range a {
		acc = acc + n
	}
	return acc
}
func Squares_plus_one(a []int) []int {
	mapped := []int{}
	for _, i := range a {
		squared := i * i
		mapped = append(mapped, squared+1)
	}
	return mapped
}
func Double_third(a []int) int {
	return a[2] * 2
}
func Last_element(a []int) int {
	return a[len(a)-1]
}
func Length_is_size(a []int) bool {
	return len(a) == len(a)
}
func Swap_positions(a bool, b int) (int, bool) {
	return b, a
}
func main() {
	mapped := []int{}
	for _, x := range []int{1, 2, 3} {
		mapped = append(mapped, x*2)
	}
	selected := []int{}
	for _, x := range mapped {
		if x > 2 {
			selected = append(selected, x)
		}
	}
	chained := selected
	arr := Make_arr(1, 2, 3)
	selected1 := []int{}
	for _, x := range Squares_plus_one([]int{1, 2, 3, 4}) {
		if x%2 == 0 {
			selected1 = append(selected1, x)
		}
	}
	qpo := len(selected1)
	total := Sum([]int{1, 2, 3, 4})
	doubled := Double_third([]int{1, 2, 3})
	last := Last_element([]int{1, 2, 3})
	foo := Length_is_size([]int{1, 2, 3})
	i, b := Swap_positions(true, 10)
}

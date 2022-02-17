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
func Length_is_size(a []int) bool {
	return len(a) == len(a)
}
func Swap_positions(a bool, b int) (int, bool) {
	return b, a
}
func main() {
	arr := Make_arr(1, 2, 3)
	selected := []int{}
	for _, x := range Squares_plus_one([]int{1, 2, 3, 4}) {
		if x%2 == 0 {
			selected = append(selected, x)
		}
	}
	qpo := len(selected)
	total := Sum([]int{1, 2, 3, 4})
	doubled := Double_third([]int{1, 2, 3})
	foo := Length_is_size([]int{1, 2, 3})
	i, b := Swap_positions(true, 10)
}

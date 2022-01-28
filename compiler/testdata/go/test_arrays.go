package main

func make_arr(a, b, c int) []int {
	arr := []int{a, b, c}
	arr = append(arr, a*b*c)
	if a > 10 {
		return []int{b}
	}
	return arr
}
func sum(a []int) int {
	acc := 0
	for _, n := range a {
		acc = acc + n
	}
	return acc
}
func squares_plus_one(a []int) []int {
	mapped := []int{}
	for _, i := range a {
		squared := i * i
		mapped = append(mapped, squared+1)
	}
	return mapped
}
func double_third(a []int) int {
	return a[2] * 2
}
func length_is_size(a []int) bool {
	return len(a) == len(a)
}
func swap_positions(a bool, b int) (int, bool) {
	return b, a
}
func main() {
	arr := make_arr(1, 2, 3)
	selected := []int{}
	for _, x := range squares_plus_one([]int{1, 2, 3, 4}) {
		if x%2 == 0 {
			selected = append(selected, x)
		}
	}
	qpo := len(selected)
	total := sum([]int{1, 2, 3, 4})
	doubled := double_third([]int{1, 2, 3})
	foo := length_is_size([]int{1, 2, 3})
	i, b := swap_positions(true, 10)
}

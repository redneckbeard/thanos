package stdlib

import (
	"math/rand"
	"reflect"
	"sort"
)

func SubtractSlice[T any](left, right []T) []T {
	for _, x := range right {
		indices := []int{}
		for i, y := range left {
			if reflect.DeepEqual(x, y) {
				indices = append([]int{i}, indices...)
			}
		}
		for _, i := range indices {
			left = append(left[:i], left[i+1:]...)
		}
	}
	return left
}

func ReverseSlice[T any](arr []T) []T {
	reversed := make([]T, len(arr))
	for i, v := range arr {
		reversed[len(arr)-1-i] = v
	}
	return reversed
}

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

func Spaceship[T Ordered](a, b T) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func Min[T Ordered](arr []T) T {
	min := arr[0]
	for _, v := range arr[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func Max[T Ordered](arr []T) T {
	max := arr[0]
	for _, v := range arr[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func Sum[T Ordered](arr []T) T {
	var sum T
	for _, v := range arr {
		sum += v
	}
	return sum
}

func Flatten[T any](arr [][]T) []T {
	result := []T{}
	for _, inner := range arr {
		result = append(result, inner...)
	}
	return result
}

func SortBy[T any, K Ordered](arr []T, key func(T) K) []T {
	sorted := make([]T, len(arr))
	copy(sorted, arr)
	sort.Slice(sorted, func(i, j int) bool {
		return key(sorted[i]) < key(sorted[j])
	})
	return sorted
}

func SortByInPlace[T any, K Ordered](arr []T, key func(T) K) {
	sort.Slice(arr, func(i, j int) bool {
		return key(arr[i]) < key(arr[j])
	})
}

func SortSlice[T Ordered](arr []T) []T {
	sorted := make([]T, len(arr))
	copy(sorted, arr)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted
}

func MinBy[T any, K Ordered](arr []T, key func(T) K) T {
	min := arr[0]
	minKey := key(min)
	for _, v := range arr[1:] {
		k := key(v)
		if k < minKey {
			min = v
			minKey = k
		}
	}
	return min
}

func MaxBy[T any, K Ordered](arr []T, key func(T) K) T {
	max := arr[0]
	maxKey := key(max)
	for _, v := range arr[1:] {
		k := key(v)
		if k > maxKey {
			max = v
			maxKey = k
		}
	}
	return max
}

func Zip[T any](a, b []T) [][]T {
	length := len(a)
	if len(b) < length {
		length = len(b)
	}
	result := make([][]T, length)
	for i := 0; i < length; i++ {
		result[i] = []T{a[i], b[i]}
	}
	return result
}

func DeleteSlice[T comparable](arr []T, val T) []T {
	result := []T{}
	for _, v := range arr {
		if v != val {
			result = append(result, v)
		}
	}
	return result
}

func DeleteAtSlice[T any](arr []T, idx int) ([]T, T) {
	if idx < 0 {
		idx = len(arr) + idx
	}
	val := arr[idx]
	return append(arr[:idx], arr[idx+1:]...), val
}

func EachCons[T any](arr []T, n int, fn func([]T)) {
	for i := 0; i <= len(arr)-n; i++ {
		fn(arr[i : i+n])
	}
}

func Intersect[T comparable](a, b []T) []T {
	set := make(map[T]bool)
	for _, v := range b {
		set[v] = true
	}
	result := []T{}
	for _, v := range a {
		if set[v] {
			result = append(result, v)
			delete(set, v)
		}
	}
	return result
}

func Union[T comparable](a, b []T) []T {
	set := make(map[T]bool)
	result := []T{}
	for _, v := range a {
		if !set[v] {
			set[v] = true
			result = append(result, v)
		}
	}
	for _, v := range b {
		if !set[v] {
			set[v] = true
			result = append(result, v)
		}
	}
	return result
}

func Rotate[T any](arr []T, n int) []T {
	if len(arr) == 0 {
		return arr
	}
	n = n % len(arr)
	if n < 0 {
		n += len(arr)
	}
	result := make([]T, len(arr))
	copy(result, arr[n:])
	copy(result[len(arr)-n:], arr[:n])
	return result
}

func Sample[T any](arr []T) T {
	return arr[rand.Intn(len(arr))]
}

func Shuffle[T any](arr []T) []T {
	shuffled := make([]T, len(arr))
	copy(shuffled, arr)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func Tally[T comparable](arr []T) *OrderedMap[T, int] {
	result := NewOrderedMap[T, int]()
	for _, v := range arr {
		if result.HasKey(v) {
			result.Set(v, result.Data[v]+1)
		} else {
			result.Set(v, 1)
		}
	}
	return result
}

func FetchSlice[T any](arr []T, idx int, defaultVal T) T {
	if idx < 0 {
		idx = len(arr) + idx
	}
	if idx < 0 || idx >= len(arr) {
		return defaultVal
	}
	return arr[idx]
}

func Clamp[T Ordered](val, min, max T) T {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func Digits(n int) []int {
	if n < 0 {
		n = -n
	}
	if n == 0 {
		return []int{0}
	}
	result := []int{}
	for n > 0 {
		result = append(result, n%10)
		n /= 10
	}
	return result
}

func Step(start, stop, step int, fn func(int)) {
	if step > 0 {
		for i := start; i <= stop; i += step {
			fn(i)
		}
	} else if step < 0 {
		for i := start; i >= stop; i += step {
			fn(i)
		}
	}
}

func TakeWhile[T any](arr []T, fn func(T) bool) []T {
	result := []T{}
	for _, v := range arr {
		if !fn(v) {
			break
		}
		result = append(result, v)
	}
	return result
}

func DropWhile[T any](arr []T, fn func(T) bool) []T {
	i := 0
	for i < len(arr) && fn(arr[i]) {
		i++
	}
	return arr[i:]
}

func Combination[T any](arr []T, n int) [][]T {
	result := [][]T{}
	if n > len(arr) {
		return result
	}
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	combo := make([]T, n)
	for i, idx := range indices {
		combo[i] = arr[idx]
	}
	c := make([]T, n)
	copy(c, combo)
	result = append(result, c)
	for {
		i := n - 1
		for i >= 0 && indices[i] == i+len(arr)-n {
			i--
		}
		if i < 0 {
			break
		}
		indices[i]++
		for j := i + 1; j < n; j++ {
			indices[j] = indices[j-1] + 1
		}
		for j, idx := range indices {
			combo[j] = arr[idx]
		}
		c := make([]T, n)
		copy(c, combo)
		result = append(result, c)
	}
	return result
}

func Permutation[T any](arr []T, n int) [][]T {
	result := [][]T{}
	if n > len(arr) {
		return result
	}
	indices := make([]int, len(arr))
	for i := range indices {
		indices[i] = i
	}
	cycles := make([]int, n)
	for i := range cycles {
		cycles[i] = len(arr) - i
	}
	perm := make([]T, n)
	for i := 0; i < n; i++ {
		perm[i] = arr[indices[i]]
	}
	p := make([]T, n)
	copy(p, perm)
	result = append(result, p)
	for {
		found := false
		for i := n - 1; i >= 0; i-- {
			cycles[i]--
			if cycles[i] == 0 {
				// Rotate indices[i:] left by 1
				saved := indices[i]
				copy(indices[i:], indices[i+1:])
				indices[len(indices)-1] = saved
				cycles[i] = len(arr) - i
			} else {
				j := len(indices) - cycles[i]
				indices[i], indices[j] = indices[j], indices[i]
				for k := 0; k < n; k++ {
					perm[k] = arr[indices[k]]
				}
				p := make([]T, n)
				copy(p, perm)
				result = append(result, p)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return result
}

func Product[T any](a, b []T) [][]T {
	result := make([][]T, 0, len(a)*len(b))
	for _, x := range a {
		for _, y := range b {
			result = append(result, []T{x, y})
		}
	}
	return result
}

func Transpose[T any](arr [][]T) [][]T {
	if len(arr) == 0 {
		return nil
	}
	cols := len(arr[0])
	result := make([][]T, cols)
	for i := range result {
		result[i] = make([]T, len(arr))
	}
	for i, row := range arr {
		for j, val := range row {
			result[j][i] = val
		}
	}
	return result
}

func Rindex[T comparable](arr []T, val T) int {
	for i := len(arr) - 1; i >= 0; i-- {
		if arr[i] == val {
			return i
		}
	}
	return -1
}

func Fill[T any](arr []T, val T) []T {
	for i := range arr {
		arr[i] = val
	}
	return arr
}

func Uniq[T comparable](arr []T) []T {
	set := make(map[T]bool)
	order := []T{}
	for _, elem := range arr {
		if _, ok := set[elem]; !ok {
			set[elem] = true
			order = append(order, elem)
		}
	}
	return order
}

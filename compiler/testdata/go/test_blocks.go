package main

type fooBlk func(b int) float64

func Foo(x, y int, blk fooBlk) float64 {
	return float64(x) * blk(y)
}
func main() {
	Foo(7, 8, func(b int) float64 {
		return float64(b) / 10.0
	})
}

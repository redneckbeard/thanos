package main

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
)

func main() {
	counts := stdlib.NewDefaultHashWithValue[string, int](0)
	counts.Set("a", counts.Get("a")+1)
	counts.Set("b", counts.Get("b")+2)
	fmt.Println(counts.Get("a"))
	fmt.Println(counts.Get("c"))
	keys := counts.Keys()
	fmt.Println(len(keys))
}

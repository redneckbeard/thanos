package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/redneckbeard/thanos/stdlib"
)

var patt = regexp.MustCompile(`good`)

func main() {
	f, _ := os.Open("stuff.txt")
	scanner := bufio.NewScanner(f)
	scanner.Split(stdlib.MakeSplitFunc("\n", false))
	for scanner.Scan() {
		ln := scanner.Text()
		subbed := patt.ReplaceAllString(ln, stdlib.ConvertFromGsub(patt, "bad"))
		fmt.Println(subbed)
	}
	f.Close()
	f1, _ := os.OpenFile("writable.txt", 522, 0666)
	f1.WriteString("here are some bits")
	info, _ := f1.Info()
	result := info.Size()
	f1.Close()
	fmt.Println(true)
}

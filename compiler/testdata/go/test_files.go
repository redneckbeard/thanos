package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
		subbed := patt.ReplaceAllString(ln, stdlib.ConvertFromGsub(

			// only here to prove that we get the return type of the block as the return type of the whole expression
			patt, "bad"))
		fmt.Println(subbed)
	}
	f.Close()

	f1, _ := os.OpenFile("writable.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f1.Close()
	f1.WriteString("here are some bits")
	info, _ := f1.Info()
	result := info.Size()
	fmt.Println(true)
	f2, _ := os.Open("readme.txt")
	defer f2.Close()
	scanner := bufio.NewScanner(f2)
	scanner.Split(stdlib.MakeSplitFunc("\n", false))
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	fmt.Println(filepath.Base("/usr/local/bin/ruby"))
	fmt.Println(filepath.Dir("/usr/local/bin/ruby"))
	fmt.Println(filepath.Ext("test.rb"))
	fmt.Println(stdlib.FileExists("stuff.txt"))
	fmt.Println(stdlib.IsDirectory("stuff"))
}

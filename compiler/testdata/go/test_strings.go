package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/redneckbeard/thanos/stdlib"
)

var patt = regexp.MustCompile(`foo`)
var patt1 = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.(?P<third>\d{1,3})\.\d{1,3}`)

func Hello(name string) string {
	fmt.Println("debug message")
	return "Hello, " + name
}
func Hello_interp(name string, age int) {
	var comparative string
	if age > 40 {
		comparative = "older"
	} else {
		comparative = "younger"
	}
	fmt.Printf("%s is %s than me, age %d\n", name, comparative, age)
}
func Matches_foo(foolike string) {
	if patt.MatchString(foolike) {
		fmt.Println("got a match")
	}
}
func Matches_interp(foo int, bar string) {
	patt, _ := regexp.Compile(fmt.Sprintf(`foo%d`, foo))
	if patt.MatchString(bar) {
		fmt.Println("got a match")
	}
}
func Extract_third_octet(ip string) string {
	return stdlib.NewMatchData(patt1, ip).GetByName("third")
}
func main() {
	greeting := Hello("me")
	Hello_interp("Steve", 38)
	Matches_foo("football")
	Matches_interp(10, "foofoo")
	Extract_third_octet("127.0.0.1")
	terms := strings.Fields(`foo bar baz`)
	interp_terms := []string{"foo", fmt.Sprintf("%s", "BAR BAZ QUUX"), "bar"}
	output, _ := exec.Command("man", "-P", "cat", fmt.Sprintf("%s", "date")).Output()
	fmt.Println(string(output))
}

package stdlib

import (
	"fmt"
	"strconv"
	"strings"
)

func Hex(s string) int {
	var hex, base int
	if !(len(s) > 2 && s[:2] == "0x") {
		base = 16
	}
	if i, err := strconv.ParseInt(s, base, 0); err == nil {
		hex = int(i)
	}
	return hex
}

func Reverse(s string) string {
	runes := []rune(s)
	i, j := 0, len(runes)-1
	for i < j {
		runes[i], runes[j] = runes[j], runes[i]
		i++
		j--
	}
	return string(runes)
}

func Join[T fmt.Stringer](t []T, delim string) string {
	segments := []string{}
	for _, segment := range t {
		segments = append(segments, segment.String())
	}
	return strings.Join(segments, delim)
}

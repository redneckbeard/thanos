package parser

import "strings"

func Indent(strs ...string) string {
	var final []string
	for _, s := range strs {
		for _, line := range strings.Split(s, "\n") {
			final = append(final, line)
		}
	}
	return strings.Join(final, "; ")
}

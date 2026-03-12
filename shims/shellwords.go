package shims

import (
	"strings"
	"unicode"
)

// ShellSplit splits a command-line string into words, respecting quotes.
// Implements Ruby's Shellwords.split / shellsplit / shellwords.
func ShellSplit(line string) []string {
	var words []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, r := range line {
		if escaped {
			if inDouble && r != '"' && r != '\\' && r != '$' && r != '`' {
				current.WriteByte('\\')
			}
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && !inSingle {
			escaped = true
			continue
		}
		if r == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if unicode.IsSpace(r) && !inSingle && !inDouble {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

// shellSpecial are characters that need escaping in shell context.
var shellSpecial = ` !"#$&'()*,:;<=>?@[\]^` + "`{|}\t\n~"

// ShellEscape escapes a string for safe use in a shell command.
// Implements Ruby's Shellwords.escape / shellescape.
func ShellEscape(str string) string {
	if str == "" {
		return "''"
	}
	var b strings.Builder
	for _, r := range str {
		if strings.ContainsRune(shellSpecial, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ShellJoin joins an array of strings into a shell-safe command.
// Implements Ruby's Shellwords.join / shelljoin.
func ShellJoin(words []string) string {
	escaped := make([]string, len(words))
	for i, w := range words {
		escaped[i] = ShellEscape(w)
	}
	return strings.Join(escaped, " ")
}

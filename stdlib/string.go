package stdlib

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func FormatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

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

func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func Ljust(s string, width int, pad string) string {
	if len(s) >= width {
		return s
	}
	for len(s) < width {
		s += pad
	}
	return s[:width]
}

func Rjust(s string, width int, pad string) string {
	if len(s) >= width {
		return s
	}
	result := ""
	for len(result)+len(s) < width {
		result += pad
	}
	return result[len(result)+len(s)-width:] + s
}

func Center(s string, width int, pad string) string {
	if len(s) >= width {
		return s
	}
	totalPad := width - len(s)
	leftPad := totalPad / 2
	rightPad := totalPad - leftPad
	left := ""
	for len(left) < leftPad {
		left += pad
	}
	left = left[:leftPad]
	right := ""
	for len(right) < rightPad {
		right += pad
	}
	right = right[:rightPad]
	return left + s + right
}

func Tr(s, from, to string) string {
	fromRunes := []rune(from)
	toRunes := []rune(to)
	mapping := make(map[rune]rune)
	for i, r := range fromRunes {
		if i < len(toRunes) {
			mapping[r] = toRunes[i]
		} else {
			mapping[r] = toRunes[len(toRunes)-1]
		}
	}
	result := []rune(s)
	for i, r := range result {
		if repl, ok := mapping[r]; ok {
			result[i] = repl
		}
	}
	return string(result)
}

func StringDelete(s, chars string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(chars, r) {
			return -1
		}
		return r
	}, s)
}

func StringBytes(s string) []int {
	bytes := []byte(s)
	result := make([]int, len(bytes))
	for i, b := range bytes {
		result[i] = int(b)
	}
	return result
}

// Squeeze removes runs of repeated characters from a string.
// If chars is empty, all repeated characters are squeezed.
// If chars is non-empty, only those characters are squeezed.
func Squeeze(s string, chars ...string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	var shouldSqueeze func(rune) bool
	if len(chars) > 0 && chars[0] != "" {
		shouldSqueeze = func(r rune) bool {
			return strings.ContainsRune(chars[0], r)
		}
	} else {
		shouldSqueeze = func(r rune) bool { return true }
	}
	result := []rune{runes[0]}
	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] && shouldSqueeze(runes[i]) {
			continue
		}
		result = append(result, runes[i])
	}
	return string(result)
}

func Swapcase(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsUpper(r) {
			return unicode.ToLower(r)
		}
		if unicode.IsLower(r) {
			return unicode.ToUpper(r)
		}
		return r
	}, s)
}

func StringSucc(s string) string {
	if len(s) == 0 {
		return ""
	}
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]
		switch {
		case r >= '0' && r < '9':
			runes[i]++
			return string(runes)
		case r == '9':
			runes[i] = '0'
			if i == 0 {
				return "1" + string(runes)
			}
		case r >= 'a' && r < 'z':
			runes[i]++
			return string(runes)
		case r == 'z':
			runes[i] = 'a'
			if i == 0 {
				return "a" + string(runes)
			}
		case r >= 'A' && r < 'Z':
			runes[i]++
			return string(runes)
		case r == 'Z':
			runes[i] = 'A'
			if i == 0 {
				return "A" + string(runes)
			}
		default:
			runes[i]++
			return string(runes)
		}
	}
	return string(runes)
}

func Oct(s string) int {
	result, _ := strconv.ParseInt(s, 8, 64)
	return int(result)
}

func Partition(s, sep string) []string {
	idx := strings.Index(s, sep)
	if idx < 0 {
		return []string{s, "", ""}
	}
	return []string{s[:idx], sep, s[idx+len(sep):]}
}

func Rpartition(s, sep string) []string {
	idx := strings.LastIndex(s, sep)
	if idx < 0 {
		return []string{"", "", s}
	}
	return []string{s[:idx], sep, s[idx+len(sep):]}
}

func Join[T fmt.Stringer](t []T, delim string) string {
	segments := []string{}
	for _, segment := range t {
		segments = append(segments, segment.String())
	}
	return strings.Join(segments, delim)
}

// StringSplice replaces s[offset:offset+length] with replacement.
// Equivalent to Ruby's s[offset, length] = replacement.
func StringSplice(s string, offset, length int, replacement string) string {
	runes := []rune(s)
	if offset < 0 {
		offset = len(runes) + offset
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(runes) {
		offset = len(runes)
	}
	end := offset + length
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[:offset]) + replacement + string(runes[end:])
}


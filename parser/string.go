package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type StringKind int

const (
	DoubleQuote StringKind = iota
	SingleQuote
	Regexp
	Words
	RawWords
	Exec
	RawExec
)

func getStringKind(delim string) StringKind {
	switch delim {
	case "\"":
		return DoubleQuote
	case "'":
		return SingleQuote
	case "/":
		return Regexp
	case "`":
		return Exec
	}
	kind := delim[1:2]
	switch kind {
	case "w":
		return RawWords
	case "W":
		return Words
	case "x":
		return RawExec
	case "X":
		return Exec
	}
	panic("The lexer should have errored already")
}

var stringDelims = map[StringKind]string{
	DoubleQuote: `"`,
	Words:       `"`,
	SingleQuote: "'",
	RawWords:    "'",
	Regexp:      "/",
	RawExec:     "`",
	Exec:        "`",
}

var validEscapes = []rune{'a', 'b', 'f', 'n', 'r', 't', 'v', '\\'}

type StringNode struct {
	BodySegments []string
	Interps      map[int][]Node
	cached       bool
	Kind         StringKind
	lineNo       int
	delim        string
	_type        types.Type
}

func (n *StringNode) OrderedInterps() []Node {
	positions := []int{}
	for k := range n.Interps {
		positions = append(positions, k)
	}
	sort.Ints(positions)
	nodes := []Node{}
	for _, i := range positions {
		interp := n.Interps[i]
		nodes = append(nodes, interp...)
	}
	return nodes
}

func (n *StringNode) GoString() string {
	switch n.Kind {
	case Regexp:
		return strings.ReplaceAll(n.FmtString("`"), "(?<", "(?P<")
	case SingleQuote, RawWords:
		return n.FmtString("`")
	default:
		return n.FmtString(`"`)
	}
}

func (n *StringNode) FmtString(delim string) string {
	if len(n.Interps) == 0 {
		if len(n.BodySegments) == 0 {
			return delim + delim
		}
		body, _ := n.TranslateEscapes(n.BodySegments[0])
		return delim + body + delim
	}
	segments := ""
	for i, seg := range n.BodySegments {
		if interps, exists := n.Interps[i]; exists {
			for _, interp := range interps {
				verb := types.FprintVerb(interp.Type())
				if verb == "" {
					panic(fmt.Sprintf("[line %d] Unhandled type inference failure for interpolated value in string", n.lineNo))
				}
				segments += verb
			}
		}
		escaped, _ := n.TranslateEscapes(seg)
		segments += escaped
	}
	if trailingInterps, exists := n.Interps[len(n.BodySegments)]; exists {
		for _, trailingInterp := range trailingInterps {
			verb := types.FprintVerb(trailingInterp.Type())
			if verb == "" {
				panic(fmt.Sprintf("[line %d] Unhandled type inference failure for interpolated value in string", n.lineNo))
			}
			segments += verb
		}
	}
	return delim + segments + delim
}

func (n *StringNode) String() string {
	if len(n.OrderedInterps()) == 0 {
		str := n.FmtString(stringDelims[n.Kind])
		if n.Kind == RawWords || n.Kind == Words {
			str = fmt.Sprintf("%%w[%s]", str)
		}
		return str
	}
	str := fmt.Sprintf(`%s %% (%s)`, n.FmtString(stringDelims[n.Kind]), stdlib.Join[Node](n.OrderedInterps(), ", "))
	if n.Kind == RawWords || n.Kind == Words {
		return fmt.Sprintf("%%w[%s]", str)
	}
	return fmt.Sprintf("(%s)", str)
}

func (n *StringNode) Type() types.Type {
	return n._type
}

func (n *StringNode) SetType(t types.Type) { n._type = t }
func (n *StringNode) LineNo() int          { return n.lineNo }

func (n *StringNode) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	if n.Kind != Regexp {
		for _, seg := range n.BodySegments {
			if _, err := n.TranslateEscapes(seg); err != nil {
				return nil, err
			}
		}
	}
	for _, interps := range n.Interps {
		for _, i := range interps {
			if t, err := GetType(i, scope, class); err != nil {
				if t == nil {
					return nil, NewParseError(n, "Could not infer type for interpolated value '%s'", i)
				}
				return nil, err
			}
		}
	}
	switch n.Kind {
	case Regexp:
		return types.RegexpType, nil
	case Words, RawWords:
		return types.NewArray(types.StringType), nil
	default:
		return types.StringType, nil
	}
}

func (n *StringNode) Copy() Node { return n }

func (n *StringNode) TranslateEscapes(segment string) (string, error) {
	switch n.Kind {
	case SingleQuote, RawWords, RawExec:
		escapeless := strings.ReplaceAll(segment, `\`+n.delim, n.delim)
		return strings.ReplaceAll(escapeless, `\\`, `\`), nil
	case DoubleQuote, Words, Exec:
		var (
			stripped []rune
			lastSeen rune
		)
		escapes := make([]rune, len(validEscapes)+1)
		copy(escapes, validEscapes)
		escapes = append(escapes, []rune(n.delim)[0])
		for _, r := range segment {
			if lastSeen == '\\' {
				for _, v := range escapes {
					if v == r {
						stripped = append(stripped, lastSeen)
					}
				}
				if r == 'e' || r == 's' {
					return "", NewParseError(n, `\%c is not a valid escape sequence in Go strings`, r)
				}
				if r == 'M' {
					return "", NewParseError(n, `\M-x, \M-\C-x, and \M-\cx are not valid escape sequences in Go strings`)
				}
				if r == 'c' || r == 'C' {
					return "", NewParseError(n, `\c\M-x, \c?, and \C? are not valid escape sequences in Go strings`)
				}
			} else if lastSeen != 0 {
				stripped = append(stripped, lastSeen)
			}
			lastSeen = r
		}
		stripped = append(stripped, lastSeen)
		return string(stripped), nil
	}
	return segment, nil
}

package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

type StringKind int

const (
	DoubleQuote StringKind = iota
	SingleQuote
	Regexp
)

var stringDelims = map[StringKind]string{
	DoubleQuote: `"`,
	SingleQuote: "'",
	Regexp:      "/",
}

type StringNode struct {
	BodySegments []string
	Interps      map[int][]Node
	cached       bool
	Kind         StringKind
	lineNo       int
}

func (n *StringNode) OrderedInterps() []Node {
	positions := []int{}
	for k, _ := range n.Interps {
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
	case SingleQuote:
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
		return delim + n.BodySegments[0] + delim
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
		segments += seg
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
	interps := []string{}
	for _, interp := range n.OrderedInterps() {
		interps = append(interps, interp.String())
	}
	if len(n.Interps) == 0 {
		return n.FmtString(stringDelims[n.Kind])
	}
	return fmt.Sprintf(`(%s %% (%s))`, n.FmtString(stringDelims[n.Kind]), strings.Join(interps, ", "))
}

func (n *StringNode) Type() types.Type {
	if len(n.Interps) == 0 || n.cached {
		switch n.Kind {
		case Regexp:
			return types.RegexpType
		default:
			return types.StringType
		}
	}
	return nil
}

func (n *StringNode) SetType(t types.Type) {}
func (n *StringNode) LineNo() int          { return n.lineNo }

func (n *StringNode) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
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
	n.cached = true
	return types.StringType, nil
}

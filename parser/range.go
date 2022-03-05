package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/types"
)

type RangeNode struct {
	Lower, Upper Node
	Inclusive    bool
	lineNo       int
	_type        types.Type
}

func (n *RangeNode) String() string {
	rangeOp := "..."
	if n.Inclusive {
		rangeOp = ".."
	}
	upper := ""
	if n.Upper != nil {
		upper = n.Upper.String()
	}
	return fmt.Sprintf("(%s%s%s)", n.Lower, rangeOp, upper)
}
func (n *RangeNode) Type() types.Type     { return n._type }
func (n *RangeNode) SetType(t types.Type) { n._type = t }
func (n *RangeNode) LineNo() int          { return n.lineNo }

func (n *RangeNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	var t types.Type
	for _, bound := range []Node{n.Lower, n.Upper} {
		if bound != nil {
			bt, err := GetType(bound, locals, class)
			if err != nil {
				return nil, err
			}
			if t != nil && t != bt {
				return nil, NewParseError(n, "Tried to construct range from disparate types %s and %s", t, bt)
			}
			t = bt
		}
	}
	return types.NewRange(t), nil
}

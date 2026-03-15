package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/types"
)

// Pos tracks the source location of an AST node.
type Pos struct {
	lineNo int
	file   string
}

func (p Pos) LineNo() int    { return p.lineNo }
func (p Pos) File() string   { return p.file }

type Node interface {
	String() string
	TargetType(ScopeChain, *Class) (types.Type, error)
	Type() types.Type
	SetType(types.Type)
	LineNo() int
	File() string
	Copy() Node
}

func GetType(n Node, scope ScopeChain, class *Class) (t types.Type, err error) {
	if Tracer != nil {
		label := nodeLabel(n)
		Tracer.Enter("GetType", label, n.LineNo())
		defer func() {
			Tracer.Exit("GetType", label, n.LineNo(), t)
		}()
	}
	t = n.Type()
	if t == nil {
		if ident, ok := n.(*IdentNode); ok {
			if loc := scope.ResolveVar(ident.Val); loc != BadLocal {
				if loc.Type() != nil {
					ident.SetType(loc.Type())
					if Tracer != nil {
						Tracer.Record("resolve-var", fmt.Sprintf("%s => %s (from scope)", ident.Val, loc.Type()))
					}
				}
				// If loc.Type() is nil, fall through to TargetType which
				// checks Kernel methods and global methods as fallbacks.
			} else if m, ok := globalMethodSet.Methods[ident.Val]; ok {
				if Tracer != nil {
					Tracer.Record("resolve-global-method", ident.Val)
				}
				if err := m.Analyze(globalMethodSet); err != nil {
					return nil, err
				}
				ident.MethodCall = &MethodCall{
					Method:     m,
					MethodName: m.Name,
					_type:      m.ReturnType(),
					Pos:        ident.Pos,
				}
				return m.ReturnType(), nil
			}
		}
		if t, err = n.TargetType(scope, class); err != nil {
			return nil, err
		} else {
			n.SetType(t)
			return t, nil
		}
	}
	return t, nil
}

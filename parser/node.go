package parser

import "github.com/redneckbeard/thanos/types"

type Node interface {
	String() string
	TargetType(ScopeChain, *Class) (types.Type, error)
	Type() types.Type
	SetType(types.Type)
	LineNo() int
}

func GetType(n Node, scope ScopeChain, class *Class) (t types.Type, err error) {
	t = n.Type()
	if t == nil {
		if ident, ok := n.(*IdentNode); ok {
			if loc := scope.ResolveVar(ident.Val); loc != BadLocal {
				ident.SetType(loc.Type())
			} else if m, ok := globalMethodSet.Methods[ident.Val]; ok {
				if err := m.Analyze(globalMethodSet); err != nil {
					return nil, err
				}
				ident.MethodCall = &MethodCall{
					Method:     m,
					MethodName: m.Name,
					_type:      m.ReturnType(),
					lineNo:     ident.lineNo,
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

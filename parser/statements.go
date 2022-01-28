package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type ArgsNode []Node

func (n ArgsNode) String() string {
	return stdlib.Join[Node](n, ", ")
}

// Wrong but dummy for satisfying interface
func (n ArgsNode) Type() types.Type     { return n[0].Type() }
func (n ArgsNode) SetType(t types.Type) {}
func (n ArgsNode) LineNo() int          { return 0 }

func (n ArgsNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	panic("ArgsNode#TargetType should never be called")
}

func (n ArgsNode) Copy() Node {
	var copy []Node
	for _, arg := range n {
		copy = append(copy, arg.Copy())
	}
	return ArgsNode(copy)
}

func (n ArgsNode) FindByName(name string) (Node, error) {
	for _, arg := range n {
		if kv, ok := arg.(*KeyValuePair); ok && kv.Label == name {
			return kv, nil
		}
	}
	return nil, fmt.Errorf("No argument named '%s' found", name)
}

type ReturnNode struct {
	Val    ArgsNode
	_type  types.Type
	lineNo int
}

func (n *ReturnNode) String() string       { return fmt.Sprintf("(return %s)", n.Val) }
func (n *ReturnNode) Type() types.Type     { return n._type }
func (n *ReturnNode) SetType(t types.Type) { n._type = t }
func (n *ReturnNode) LineNo() int          { return n.lineNo }

func (n *ReturnNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if len(n.Val) == 1 {
		return GetType(n.Val[0], locals, class)
	}
	multiple := types.Multiple{}
	for _, single := range n.Val {
		t, err := GetType(single, locals, class)
		if err != nil {
			return t, err
		}
		multiple = append(multiple, t)
	}
	return multiple, nil
}

func (n *ReturnNode) Copy() Node {
	return &ReturnNode{n.Val.Copy().(ArgsNode), n._type, n.lineNo}
}

type Statements []Node

func (stmts Statements) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	var lastReturnedType types.Type
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *AssignmentNode:
			if t, err := GetType(s, scope, class); err != nil {
				return nil, err
			} else {
				lastReturnedType = t
			}
		case *Condition:
			// We need this to be semi-"live" since otherwise we can't surface an
			// error about a type mismatch between the branches. The type on the
			// condition will still be effectively memoized since it can just get the
			// cached value from the True side. Thus we call TargetType directly on
			// the node instead of going through GetType.
			if t, err := GetType(s, scope, class); err != nil {
				return nil, err
			} else {
				lastReturnedType = t
			}
		case *IVarNode:
			if t, err := GetType(s, scope, class); err != nil {
				return nil, err
			} else {
				lastReturnedType = t
			}
		default:
			if c, ok := stmt.(*MethodCall); ok {
				// Handle method chaining -- walk down to the first identifier or
				// literal and infer types on the way back up so that receiver type is
				// known for each subsequent method call
				chain := []Node{c}
				r := c.Receiver
				walking := true
				for walking {
					switch c := r.(type) {
					case *MethodCall:
						chain = append(chain, c)
						r = c.Receiver
					default:
						if c != nil {
							chain = append(chain, c)
						}
						walking = false
					}
				}
				for i := len(chain) - 1; i >= 0; i-- {
					if t, err := GetType(chain[i], scope, class); err != nil {
						return nil, err
					} else {
						lastReturnedType = t
					}
				}
			} else if t, err := GetType(stmt, scope, class); err != nil {
				return nil, err
			} else {
				lastReturnedType = t
			}
		}
	}
	return lastReturnedType, nil
}

func (stmts Statements) String() string {
	switch len(stmts) {
	case 0:
		return ""
	case 1:
		return stmts[0].String()
	default:
		return fmt.Sprintf("%s", stdlib.Join[Node](stmts, "\n"))
	}
}

func (stmts Statements) Type() types.Type     { return nil }
func (stmts Statements) SetType(t types.Type) {}
func (stmts Statements) LineNo() int          { return 0 }

func (stmts Statements) Copy() Node {
	var copy []Node
	for _, stmt := range stmts {
		copy = append(copy, stmt.Copy())
	}
	return Statements(stmts)
}

type Body struct {
	Statements      Statements
	ReturnType      types.Type
	ExplicitReturns []*ReturnNode
}

func (b *Body) InferReturnType(scope ScopeChain, class *Class) error {
	// To guess the right return type of a method, we have to:

	//	1) track all return statements in the method body;

	//  2) chase expressions all the way to the end of the body and wrap that
	//  last expr in a return node if it's not already there, wherein we record
	//  the types of all assignments in a map on the method.

	// Achieving 1) would mean rewalking this branch of the AST right after
	// building it which seems dumb, so instead we register each ReturnNode on
	// the method as the parser encounters them so we can loop through them
	// afterward when m.Locals is fully populated.

	lastReturnedType, err := GetType(b.Statements, scope, class)
	if err != nil {
		return err
	}
	finalStatementIdx := len(b.Statements) - 1
	finalStatement := b.Statements[finalStatementIdx]
	switch s := finalStatement.(type) {
	case *ReturnNode:
	case *AssignmentNode:
		var ret *ReturnNode
		if s.OpAssignment {
			ret = &ReturnNode{Val: s.Left}
		} else if _, ok := s.Left[0].(*IVarNode); ok {
			ret = &ReturnNode{Val: s.Left}
		} else {
			ret = &ReturnNode{Val: []Node{s.Right[0]}}
		}
		if _, err := GetType(ret, scope, class); err != nil {
			return err
		}
		b.Statements = append(b.Statements, ret)
	default:
		if finalStatement.Type() != types.NilType && scope.Name() != Main {
			ret := &ReturnNode{Val: []Node{finalStatement}}
			if _, err := GetType(ret, scope, class); err != nil {
				return err
			}
			b.Statements[finalStatementIdx] = ret
		}
	}
	if len(b.ExplicitReturns) > 0 {
		for _, r := range b.ExplicitReturns {
			t, _ := GetType(r, scope, class)
			if !t.Equals(lastReturnedType) {
				return NewParseError(r, "Detected conflicting return types %s and %s in method '%s'", lastReturnedType, t, scope.Name())
			}
		}
	}
	b.ReturnType = lastReturnedType
	return nil
}

func (n *Body) String() string {
	return n.Statements.String()
}

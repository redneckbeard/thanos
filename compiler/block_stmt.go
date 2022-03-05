package compiler

import (
	"go/ast"

	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

func (g *GoProgram) CompileBlockStmt(node parser.Node) *ast.BlockStmt {
	blockStmt := g.newBlockStmt()
	defer g.popBlockStmt()
	switch n := node.(type) {
	case *parser.Condition:
		if n.False != nil {
			g.CompileStmt(n)
			return blockStmt
		} else {
			return g.CompileBlockStmt(n.True)
		}
	case parser.Statements:
		last := len(n) - 1
		for i, s := range n {
			if i == last {
				if g.CurrentState() == InReturnStatement && s.Type() != types.NilType {
					t := s.Type()
					s = &parser.ReturnNode{Val: parser.ArgsNode{s}}
					s.SetType(t)
				} else if g.CurrentState() == InCondAssignment {
					s = &parser.AssignmentNode{
						Left:         g.CurrentLhs,
						Right:        []parser.Node{s},
						Reassignment: true,
					}
				}
			}
			g.CompileStmt(s)
		}
		return blockStmt
	default:
		return &ast.BlockStmt{}
	}
}

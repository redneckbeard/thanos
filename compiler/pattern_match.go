package compiler

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

func (g *GoProgram) compilePatternMatch(n *parser.PatternMatchNode) {
	// When the subject is a tuple (heterogeneous array literal), we can't compile
	// it as a single Go expression. Instead, we compile each element separately and
	// use a tupleSubject to give pattern elements direct access.
	var subject ast.Expr
	var tupleExprs []ast.Expr
	if arrNode, ok := n.Value.(*parser.ArrayNode); ok {
		if _, isTuple := arrNode.Type().(*types.Tuple); isTuple {
			for _, elem := range arrNode.Args {
				tupleExprs = append(tupleExprs, g.CompileExpr(elem))
			}
		}
	}
	if tupleExprs == nil {
		subject = g.CompileExpr(n.Value)
	}

	// Build an if-else chain for each in clause
	var rootIf *ast.IfStmt
	var lastIf *ast.IfStmt

	for _, clause := range n.InClauses {
		ifStmt := &ast.IfStmt{}

		// Collect used identifiers from clause body to avoid unused variable errors
		usedIdents := parser.CollectIdents(clause.Statements)

		// Generate condition and bindings for the pattern
		var cond ast.Expr
		var bindings []ast.Stmt
		if tupleExprs != nil {
			cond, bindings = g.compileTuplePatternCondition(clause.Pattern, tupleExprs, usedIdents)
		} else {
			cond, bindings = g.compilePatternCondition(clause.Pattern, subject, n.Value, usedIdents)
		}
		ifStmt.Cond = cond

		// Body: bindings + compiled statements
		body := &ast.BlockStmt{}
		body.List = append(body.List, bindings...)
		body.List = append(body.List, g.CompileBlockStmt(clause.Statements).List...)
		ifStmt.Body = body

		if rootIf == nil {
			rootIf = ifStmt
		} else {
			lastIf.Else = ifStmt
		}
		lastIf = ifStmt
	}

	// Else clause
	if n.ElseBody != nil && rootIf != nil {
		lastIf.Else = g.CompileBlockStmt(n.ElseBody)
	}

	if rootIf != nil {
		g.appendToCurrentBlock(rootIf)
	}
}

// compilePatternCondition generates a Go condition expression and variable
// bindings for a pattern match clause.
func (g *GoProgram) compilePatternCondition(pattern parser.Node, subject ast.Expr, valueNode parser.Node, usedIdents map[string]bool) (ast.Expr, []ast.Stmt) {
	switch p := pattern.(type) {
	case *parser.ArrayPatternNode:
		return g.compileArrayPattern(p, subject, valueNode, usedIdents)
	case *parser.IdentNode:
		// Variable capture — always matches, bind the variable
		if !usedIdents[p.Val] {
			return g.it.Get("true"), nil
		}
		ident := g.it.Get(p.Val)
		binding := bst.Define(ident, subject)
		return g.it.Get("true"), []ast.Stmt{binding}
	case *parser.WildcardPatternNode:
		// Always matches, no binding
		return g.it.Get("true"), nil
	case *parser.NilNode:
		return bst.Binary(subject, token.EQL, g.it.Get("nil")), nil
	case *parser.BooleanNode:
		return bst.Binary(subject, token.EQL, g.it.Get(p.Val)), nil
	case *parser.IntNode:
		return bst.Binary(subject, token.EQL, g.CompileExpr(p)), nil
	case *parser.StringNode:
		return bst.Binary(subject, token.EQL, g.CompileExpr(p)), nil
	case *parser.ArrayNode:
		// Empty array literal in pattern: check len == 0
		if len(p.Args) == 0 {
			return bst.Binary(
				bst.Call(nil, "len", subject),
				token.EQL,
				bst.Int("0"),
			), nil
		}
		// Non-empty literal array — check equality
		return bst.Binary(subject, token.EQL, g.CompileExpr(p)), nil
	default:
		// Fallback: direct equality
		return bst.Binary(subject, token.EQL, g.CompileExpr(pattern)), nil
	}
}

func (g *GoProgram) compileArrayPattern(p *parser.ArrayPatternNode, subject ast.Expr, valueNode parser.Node, usedIdents map[string]bool) (ast.Expr, []ast.Stmt) {
	var conds []ast.Expr
	var bindings []ast.Stmt

	// Length check: len(subject) == len(p.Elements)
	lenCheck := bst.Binary(
		bst.Call(nil, "len", subject),
		token.EQL,
		bst.Int(itoa(len(p.Elements))),
	)
	conds = append(conds, lenCheck)

	// For each element, generate index access and sub-pattern conditions
	for i, elem := range p.Elements {
		indexExpr := &ast.IndexExpr{
			X:     subject,
			Index: bst.Int(itoa(i)),
		}
		switch e := elem.(type) {
		case *parser.IdentNode:
			if e.Val != "_" && usedIdents[e.Val] {
				ident := g.it.Get(e.Val)
				bindings = append(bindings, bst.Define(ident, indexExpr))
			}
		case *parser.WildcardPatternNode:
			// No binding
		case *parser.ArrayPatternNode:
			// Nested array pattern — add length check and recurse
			nestedCond, nestedBindings := g.compileArrayPattern(e, indexExpr, valueNode, usedIdents)
			conds = append(conds, nestedCond)
			bindings = append(bindings, nestedBindings...)
		case *parser.ArrayNode:
			// Empty array literal — check len == 0
			if len(e.Args) == 0 {
				conds = append(conds, bst.Binary(
					bst.Call(nil, "len", indexExpr),
					token.EQL,
					bst.Int("0"),
				))
			} else {
				subCond, subBindings := g.compilePatternCondition(e, indexExpr, valueNode, usedIdents)
				conds = append(conds, subCond)
				bindings = append(bindings, subBindings...)
			}
		default:
			// Value pattern — equality check
			subCond, subBindings := g.compilePatternCondition(e, indexExpr, valueNode, usedIdents)
			conds = append(conds, subCond)
			bindings = append(bindings, subBindings...)
		}
	}

	// Combine all conditions with &&
	combined := conds[0]
	for _, c := range conds[1:] {
		combined = bst.Binary(combined, token.LAND, c)
	}

	return combined, bindings
}

// compileTuplePatternCondition handles pattern matching when the subject is a
// tuple (heterogeneous array literal). Each element is accessed directly by index.
func (g *GoProgram) compileTuplePatternCondition(pattern parser.Node, tupleExprs []ast.Expr, usedIdents map[string]bool) (ast.Expr, []ast.Stmt) {
	arrPat, ok := pattern.(*parser.ArrayPatternNode)
	if !ok {
		// If it's not an array pattern, fall through to wildcard/ident logic
		if _, ok := pattern.(*parser.WildcardPatternNode); ok {
			return g.it.Get("true"), nil
		}
		if ident, ok := pattern.(*parser.IdentNode); ok {
			if !usedIdents[ident.Val] {
				return g.it.Get("true"), nil
			}
		}
		return g.it.Get("true"), nil
	}

	var conds []ast.Expr
	var bindings []ast.Stmt

	// Length must match
	if len(arrPat.Elements) != len(tupleExprs) {
		return g.it.Get("false"), nil
	}

	for i, elem := range arrPat.Elements {
		subjectElem := tupleExprs[i]
		switch e := elem.(type) {
		case *parser.IdentNode:
			if e.Val != "_" && usedIdents[e.Val] {
				bindings = append(bindings, bst.Define(g.it.Get(e.Val), subjectElem))
			}
		case *parser.WildcardPatternNode:
			// No binding, no condition
		case *parser.ArrayPatternNode:
			// Nested array pattern — use the regular array pattern compiler
			nestedCond, nestedBindings := g.compileArrayPattern(e, subjectElem, nil, usedIdents)
			conds = append(conds, nestedCond)
			bindings = append(bindings, nestedBindings...)
		case *parser.ArrayNode:
			if len(e.Args) == 0 {
				conds = append(conds, bst.Binary(
					bst.Call(nil, "len", subjectElem),
					token.EQL,
					bst.Int("0"),
				))
			}
		default:
			subCond, subBindings := g.compilePatternCondition(e, subjectElem, nil, usedIdents)
			conds = append(conds, subCond)
			bindings = append(bindings, subBindings...)
		}
	}

	if len(conds) == 0 {
		return g.it.Get("true"), bindings
	}
	combined := conds[0]
	for _, c := range conds[1:] {
		combined = bst.Binary(combined, token.LAND, c)
	}
	return combined, bindings
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

package compiler

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"github.com/redneckbeard/thanos/parser"
)

// commentState tracks comment emission during compilation.
type commentState struct {
	fset     *token.FileSet
	file     *token.File
	goLine   int                    // current Go output line counter
	comments []*ast.CommentGroup    // collected comment groups for ast.File
	pending  []parser.Comment       // Ruby comments sorted by line number
	stmtLines map[ast.Stmt]int     // Go statement → Ruby line number
}

func newCommentState(rubyComments map[int]parser.Comment) *commentState {
	fset := token.NewFileSet()
	file := fset.AddFile("main.go", -1, 1000000)
	lines := make([]int, 100000)
	for i := range lines {
		lines[i] = i * 10
	}
	file.SetLines(lines)

	// Sort comments by line number
	sorted := make([]parser.Comment, 0, len(rubyComments))
	for _, c := range rubyComments {
		sorted = append(sorted, c)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LineNo < sorted[j].LineNo
	})

	return &commentState{
		fset:      fset,
		file:      file,
		goLine:    4, // start after package + imports (adjusted later)
		pending:   sorted,
		stmtLines: make(map[ast.Stmt]int),
	}
}

// pos returns a token.Pos for the given Go output line.
func (cs *commentState) pos(line int) token.Pos {
	if line < 1 {
		line = 1
	}
	if line >= 100000 {
		line = 99999
	}
	return cs.file.LineStart(line)
}

// nextLine advances the Go line counter and returns a pos for the current line.
func (cs *commentState) nextLine() token.Pos {
	cs.goLine++
	return cs.pos(cs.goLine)
}

// emitCommentsBefore checks for Ruby comments with line numbers up to (but not
// including) rubyLine, and returns them as Go comment groups positioned before
// the next statement. Each emitted comment advances the Go line counter.
func (cs *commentState) emitCommentsBefore(rubyLine int) {
	for len(cs.pending) > 0 && cs.pending[0].LineNo < rubyLine {
		c := cs.pending[0]
		cs.pending = cs.pending[1:]

		cs.goLine++
		cg := &ast.CommentGroup{
			List: []*ast.Comment{
				{Slash: cs.pos(cs.goLine), Text: rubyToGoComment(c.Text)},
			},
		}
		cs.comments = append(cs.comments, cg)
	}
}

// emitRemainingComments emits any comments that haven't been placed yet
// (e.g., comments at the end of the file after all code).
func (cs *commentState) emitRemainingComments() {
	for _, c := range cs.pending {
		cs.goLine++
		cg := &ast.CommentGroup{
			List: []*ast.Comment{
				{Slash: cs.pos(cs.goLine), Text: rubyToGoComment(c.Text)},
			},
		}
		cs.comments = append(cs.comments, cg)
	}
	cs.pending = nil
}

// rubyToGoComment converts a Ruby comment (starting with #) to a Go comment
// (starting with //).
func rubyToGoComment(text string) string {
	text = strings.TrimPrefix(text, "#")
	if len(text) > 0 && text[0] != ' ' {
		text = " " + text
	}
	return "//" + text
}

// setStmtPos sets only the leading position token of a statement, without
// recursing into sub-expressions. This avoids clobbering shared ident pointers
// (IdentTracker.Get returns the same *ast.Ident for the same name).
func setStmtPos(stmt ast.Stmt, p token.Pos) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		s.TokPos = p
		// Clone the first LHS to avoid clobbering shared idents
		if len(s.Lhs) > 0 {
			s.Lhs[0] = setExprLeadingPos(s.Lhs[0], p)
		}
	case *ast.ExprStmt:
		s.X = setExprLeadingPos(s.X, p)
	case *ast.IfStmt:
		s.If = p
	case *ast.ForStmt:
		s.For = p
	case *ast.RangeStmt:
		s.For = p
	case *ast.SwitchStmt:
		s.Switch = p
	case *ast.TypeSwitchStmt:
		s.Switch = p
	case *ast.ReturnStmt:
		s.Return = p
	case *ast.DeclStmt:
		if gd, ok := s.Decl.(*ast.GenDecl); ok {
			gd.TokPos = p
		}
	case *ast.IncDecStmt:
		s.TokPos = p
	case *ast.BranchStmt:
		s.TokPos = p
	case *ast.DeferStmt:
		s.Defer = p
	case *ast.GoStmt:
		s.Go = p
	}
}

// setExprLeadingPos sets the leading position of an expression by cloning
// the first ident to avoid clobbering shared IdentTracker pointers.
func setExprLeadingPos(expr ast.Expr, p token.Pos) ast.Expr {
	switch e := expr.(type) {
	case *ast.CallExpr:
		e.Fun = setExprLeadingPos(e.Fun, p)
		e.Lparen = p
		e.Rparen = p
	case *ast.SelectorExpr:
		e.X = setExprLeadingPos(e.X, p)
	case *ast.Ident:
		return &ast.Ident{Name: e.Name, NamePos: p}
	case *ast.BasicLit:
		return &ast.BasicLit{Kind: e.Kind, Value: e.Value, ValuePos: p}
	case *ast.UnaryExpr:
		e.OpPos = p
	case *ast.ParenExpr:
		e.Lparen = p
	case *ast.CompositeLit:
		e.Lbrace = p
		e.Rbrace = p
	}
	return expr
}

// stampStmt assigns a Go line to a statement and recursively handles
// compound statements so their blocks span multiple lines.
func (cs *commentState) stampStmt(stmt ast.Stmt) {
	cs.goLine++
	p := cs.pos(cs.goLine)

	setStmtPos(stmt, p)

	switch s := stmt.(type) {
	case *ast.IfStmt:
		s.Body.Lbrace = p
		cs.stampBlock(s.Body)
		if s.Else != nil {
			switch e := s.Else.(type) {
			case *ast.BlockStmt:
				e.Lbrace = cs.pos(cs.goLine)
				cs.stampBlock(e)
			case *ast.IfStmt:
				cs.stampStmt(e)
			}
		}
	case *ast.ForStmt:
		s.Body.Lbrace = p
		cs.stampBlock(s.Body)
	case *ast.RangeStmt:
		s.Body.Lbrace = p
		cs.stampBlock(s.Body)
	case *ast.SwitchStmt:
		s.Body.Lbrace = p
		cs.stampBlock(s.Body)
	case *ast.TypeSwitchStmt:
		s.Body.Lbrace = p
		cs.stampBlock(s.Body)
	}
}

// stampBlock positions each statement in a block and sets the closing brace.
func (cs *commentState) stampBlock(block *ast.BlockStmt) {
	for _, stmt := range block.List {
		cs.stampStmt(stmt)
	}
	cs.goLine++
	block.Rbrace = cs.pos(cs.goLine)
}

// stampBlockWithComments positions statements and interleaves Ruby comments
// based on the Ruby line numbers associated with each Go statement.
func (cs *commentState) stampBlockWithComments(block *ast.BlockStmt) {
	for _, stmt := range block.List {
		// Emit any comments that precede this statement's Ruby line
		if rubyLine, ok := cs.stmtLines[stmt]; ok {
			cs.emitCommentsBefore(rubyLine)
		}
		cs.stampStmt(stmt)
	}
	// Emit any remaining comments after the last statement
	cs.emitRemainingComments()
	cs.goLine++
	block.Rbrace = cs.pos(cs.goLine)
}

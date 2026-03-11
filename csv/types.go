package csv

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/types"
)

var csvImport = "github.com/redneckbeard/thanos/csv"

func init() {
	// Register the csvWriter sentinel type so generate/open can look it up.
	types.RegisterNamedType("CSVWriter", &csvWriterImpl{})

	// CSV.read(filename, headers: bool, col_sep: string) -> [][]string or *csv.Table
	types.CSVClass.Def("read", types.MethodSpec{
		KwargsSpec: []types.KwargSpec{
			{Name: "headers", Type: types.BoolType},
			{Name: "col_sep", Type: types.StringType},
		},
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			if len(args) > 1 && args[1] != nil {
				return csvTableType(), nil
			}
			return types.NewArray(types.NewArray(types.StringType)), nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			hasHeaders := len(args) > 1 && args[1].Expr != nil
			hasColSep := len(args) > 2 && args[2].Expr != nil
			if hasHeaders && hasColSep {
				return types.Transform{
					Expr:    bst.Call("csv", "ReadWithHeadersAndOptions", args[0].Expr, colSepRune(args[2].Expr)),
					Imports: []string{csvImport},
				}
			} else if hasHeaders {
				return types.Transform{
					Expr:    bst.Call("csv", "ReadWithHeaders", args[0].Expr),
					Imports: []string{csvImport},
				}
			} else if hasColSep {
				return types.Transform{
					Expr:    bst.Call("csv", "ReadWithOptions", args[0].Expr, colSepRune(args[2].Expr)),
					Imports: []string{csvImport},
				}
			}
			return types.Transform{
				Expr:    bst.Call("csv", "Read", args[0].Expr),
				Imports: []string{csvImport},
			}
		},
	})

	// CSV.readlines(filename) -> alias for read
	types.CSVClass.MakeAlias("read", "readlines", true)

	// CSV.parse(string, headers: bool, col_sep: string) -> [][]string or *csv.Table
	types.CSVClass.Def("parse", types.MethodSpec{
		KwargsSpec: []types.KwargSpec{
			{Name: "headers", Type: types.BoolType},
			{Name: "col_sep", Type: types.StringType},
		},
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			if len(args) > 1 && args[1] != nil {
				return csvTableType(), nil
			}
			return types.NewArray(types.NewArray(types.StringType)), nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			hasHeaders := len(args) > 1 && args[1].Expr != nil
			hasColSep := len(args) > 2 && args[2].Expr != nil
			if hasHeaders && hasColSep {
				return types.Transform{
					Expr:    bst.Call("csv", "ParseWithHeadersAndOptions", args[0].Expr, colSepRune(args[2].Expr)),
					Imports: []string{csvImport},
				}
			} else if hasHeaders {
				return types.Transform{
					Expr:    bst.Call("csv", "ParseWithHeaders", args[0].Expr),
					Imports: []string{csvImport},
				}
			} else if hasColSep {
				return types.Transform{
					Expr:    bst.Call("csv", "ParseWithOptions", args[0].Expr, colSepRune(args[2].Expr)),
					Imports: []string{csvImport},
				}
			}
			return types.Transform{
				Expr:    bst.Call("csv", "Parse", args[0].Expr),
				Imports: []string{csvImport},
			}
		},
	})

	// CSV.foreach(filename, headers: bool, col_sep: string) { |row| ... }
	types.CSVClass.Def("foreach", types.MethodSpec{
		KwargsSpec: []types.KwargSpec{
			{Name: "headers", Type: types.BoolType},
			{Name: "col_sep", Type: types.StringType},
		},
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return types.NilType, nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			types.StripBlockReturn(blk)
			types.BlankUnusedBlockArgs(blk)

			hasHeaders := len(args) > 1 && args[1].Expr != nil
			hasColSep := len(args) > 2 && args[2].Expr != nil

			if hasHeaders {
				return foreachWithHeaders(args, blk, it, hasColSep)
			}
			return foreachPlain(args, blk, it, hasColSep)
		},
	})
	// Set blockArgs separately since it's unexported
	types.CSVClass.SetBlockArgs("foreach", func(r types.Type, args []types.Type) []types.Type {
		if len(args) > 1 && args[1] != nil {
			return []types.Type{csvTableRowType()}
		}
		return []types.Type{types.NewArray(types.StringType)}
	})

	// CSV.generate { |csv| csv << ["a","b"] } -> string
	types.CSVClass.Def("generate", types.MethodSpec{
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return types.StringType, nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			types.StripBlockReturn(blk)

			buf := it.New("buf")
			writer := it.New("writer")

			declBuf := &ast.DeclStmt{
				Decl: bst.Declare(token.VAR, buf, bst.Dot("strings", "Builder")),
			}
			newWriter := bst.Define(writer, bst.Call("csv", "NewWriter", &ast.UnaryExpr{Op: token.AND, X: buf}))
			rewriteIdent(blk.Statements, blk.Args[0].(*ast.Ident).Name, writer)
			flush := &ast.ExprStmt{X: bst.Call(writer, "Flush")}

			stmts := []ast.Stmt{declBuf, newWriter}
			stmts = append(stmts, blk.Statements...)
			stmts = append(stmts, flush)

			return types.Transform{
				Stmts:   stmts,
				Expr:    bst.Call(buf, "String"),
				Imports: []string{"strings", "encoding/csv"},
			}
		},
	})
	types.CSVClass.SetBlockArgs("generate", func(r types.Type, args []types.Type) []types.Type {
		return []types.Type{csvWriterType()}
	})

	// CSV.open(filename, mode) { |csv| csv << row }
	types.CSVClass.Def("open", types.MethodSpec{
		ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
			return types.NilType, nil
		},
		TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
			types.StripBlockReturn(blk)

			f := it.New("f")
			writer := it.New("writer")

			openFile := bst.Define(
				[]ast.Expr{f, ast.NewIdent("_")},
				bst.Call("os", "Create", args[0].Expr),
			)
			deferClose := &ast.DeferStmt{Call: bst.Call(f, "Close")}
			newWriter := bst.Define(writer, bst.Call("csv", "NewWriter", f))
			rewriteIdent(blk.Statements, blk.Args[0].(*ast.Ident).Name, writer)
			flush := &ast.ExprStmt{X: bst.Call(writer, "Flush")}

			stmts := []ast.Stmt{openFile, deferClose, newWriter}
			stmts = append(stmts, blk.Statements...)
			stmts = append(stmts, flush)

			return types.Transform{
				Stmts:   stmts,
				Imports: []string{"os", "encoding/csv"},
			}
		},
	})
	types.CSVClass.SetBlockArgs("open", func(r types.Type, args []types.Type) []types.Type {
		return []types.Type{csvWriterType()}
	})
}

// csvTableType returns the facade-defined CSV::Table type from the registry.
func csvTableType() types.Type {
	t, _ := types.LookupNamedType("CSV::Table")
	return t
}

// csvTableRowType returns the facade-defined CSV::Row type.
func csvTableRowType() types.Type {
	t, _ := types.LookupNamedType("CSV::Row")
	return t
}

// csvWriterType returns the csvWriter sentinel type from the registry.
func csvWriterType() types.Type {
	t, _ := types.LookupNamedType("CSVWriter")
	return t
}

// foreachPlain generates the for-loop pattern for CSV.foreach without headers.
func foreachPlain(args []types.TypeExpr, blk *types.Block, it bst.IdentTracker, hasColSep bool) types.Transform {
	f := it.New("f")
	reader := it.New("reader")
	err := it.New("err")

	openFile := bst.Define(
		[]ast.Expr{f, ast.NewIdent("_")},
		bst.Call("os", "Open", args[0].Expr),
	)
	deferClose := &ast.DeferStmt{Call: bst.Call(f, "Close")}
	newReader := bst.Define(reader, bst.Call("csv", "NewReader", f))

	var stmts []ast.Stmt
	stmts = append(stmts, openFile, deferClose, newReader)

	if hasColSep {
		stmts = append(stmts, &ast.AssignStmt{
			Lhs: []ast.Expr{bst.Dot(reader, "Comma")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{colSepRune(args[2].Expr)},
		})
	}

	readRow := bst.Define(
		[]ast.Expr{blk.Args[0], err},
		bst.Call(reader, "Read"),
	)
	breakOnErr := &ast.IfStmt{
		Cond: bst.Binary(err, token.NEQ, ast.NewIdent("nil")),
		Body: &ast.BlockStmt{
			List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}},
		},
	}
	loop := &ast.ForStmt{
		Body: &ast.BlockStmt{
			List: append([]ast.Stmt{readRow, breakOnErr}, blk.Statements...),
		},
	}
	stmts = append(stmts, loop)

	return types.Transform{
		Stmts:   stmts,
		Imports: []string{"os", "encoding/csv"},
	}
}

// foreachWithHeaders generates a for-range loop over csv.ReadWithHeaders.
func foreachWithHeaders(args []types.TypeExpr, blk *types.Block, it bst.IdentTracker, hasColSep bool) types.Transform {
	table := it.New("table")

	var readExpr ast.Expr
	if hasColSep {
		readExpr = bst.Call("csv", "ReadWithHeadersAndOptions", args[0].Expr, colSepRune(args[2].Expr))
	} else {
		readExpr = bst.Call("csv", "ReadWithHeaders", args[0].Expr)
	}

	tableDecl := bst.Define(table, readExpr)
	loop := &ast.RangeStmt{
		Key:   ast.NewIdent("_"),
		Value: blk.Args[0],
		Tok:   token.DEFINE,
		X:     bst.Call(table, "ToSlice"),
		Body:  &ast.BlockStmt{List: blk.Statements},
	}

	return types.Transform{
		Stmts:   []ast.Stmt{tableDecl, loop},
		Imports: []string{csvImport},
	}
}

// colSepRune converts a string expression to a rune: []rune(s)[0]
func colSepRune(expr ast.Expr) ast.Expr {
	return &ast.IndexExpr{
		X: &ast.CallExpr{
			Fun:  &ast.ArrayType{Elt: ast.NewIdent("rune")},
			Args: []ast.Expr{expr},
		},
		Index: bst.Int(0),
	}
}

// rewriteIdent walks an AST statement list and renames all occurrences
// of oldName to point at the newIdent.
func rewriteIdent(stmts []ast.Stmt, oldName string, newIdent *ast.Ident) {
	ast.Inspect(&ast.BlockStmt{List: stmts}, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == oldName {
			id.Name = newIdent.Name
		}
		return true
	})
}

// --- csvWriterImpl: sentinel type for CSV.generate / CSV.open block parameter ---

type csvWriterImpl struct{}

func (t *csvWriterImpl) Equals(t2 types.Type) bool    { return t == t2 }
func (t *csvWriterImpl) String() string                { return "CSVWriter" }
func (t *csvWriterImpl) GoType() string                { return "*csv.Writer" }
func (t *csvWriterImpl) IsComposite() bool             { return false }
func (t *csvWriterImpl) IsMultiple() bool              { return false }
func (t *csvWriterImpl) ClassName() string             { return "CSVWriter" }
func (t *csvWriterImpl) BlockArgTypes(m string, args []types.Type) []types.Type { return nil }

func (t *csvWriterImpl) MethodReturnType(m string, b types.Type, args []types.Type) (types.Type, error) {
	if m == "<<" {
		return t, nil
	}
	return types.NilType, nil
}

func (t *csvWriterImpl) HasMethod(m string) bool { return m == "<<" }

func (t *csvWriterImpl) GetMethodSpec(m string) (types.MethodSpec, bool) {
	if m == "<<" {
		return types.MethodSpec{
			ReturnType: func(r types.Type, b types.Type, args []types.Type) (types.Type, error) {
				return r, nil
			},
			TransformAST: func(rcvr types.TypeExpr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
				return types.Transform{
					Expr: bst.Call(rcvr.Expr, "Write", args[0].Expr),
				}
			},
		}, true
	}
	return types.MethodSpec{}, false
}

func (t *csvWriterImpl) TransformAST(m string, rcvr ast.Expr, args []types.TypeExpr, blk *types.Block, it bst.IdentTracker) types.Transform {
	if spec, ok := t.GetMethodSpec(m); ok {
		return spec.TransformAST(types.TypeExpr{Type: t, Expr: rcvr}, args, blk, it)
	}
	return types.Transform{}
}

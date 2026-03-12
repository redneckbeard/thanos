package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

func FprintVerb(t Type) string {
	switch t {
	case StringType:
		return "%s"
	case IntType:
		return "%d"
	case FloatType:
		return "%f"
	case BoolType:
		return "%b"
	case nil:
		return ""
	default:
		return "%v"
	}
}

type String struct {
	*proto
}

var StringType = String{newProto("String", "Object", ClassRegistry)}

var StringClass = NewClass("String", "Object", StringType, ClassRegistry)

func (t String) Equals(t2 Type) bool { return t == t2 }
func (t String) String() string      { return "StringType" }
func (t String) GoType() string      { return "string" }
func (t String) IsComposite() bool   { return false }

func (t String) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t String) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t String) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t String) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t String) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t String) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t String) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
	// `String#%`
	StringType.Def("+", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return receiverType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.ADD, args[0].Expr),
			}
		},
	})
	StringType.Alias("+", "<<")
	StringType.Def("<", simpleComparisonOperatorSpec(token.LSS))
	StringType.Def(">", simpleComparisonOperatorSpec(token.GTR))
	StringType.Def("<=", simpleComparisonOperatorSpec(token.LEQ))
	StringType.Def(">=", simpleComparisonOperatorSpec(token.GEQ))
	StringType.Def("==", simpleComparisonOperatorSpec(token.EQL))
	StringType.Def("!=", simpleComparisonOperatorSpec(token.NEQ))
	StringType.Def("<=>", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Spaceship", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("=~", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			// In reality the match operator returns an int, or nil if there's no match. However, in practical
			// use it is relied on for evaluation to a boolean
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(args[0].Expr, "MatchString", rcvr.Expr),
			}
		},
	})
	StringType.Def("*", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "Repeat", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("%", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			fmtArgs := []ast.Expr{rcvr.Expr}
			// Array literal RHS: spread elements as individual Sprintf args
			if arrLit, ok := args[0].Expr.(*ast.CompositeLit); ok {
				fmtArgs = append(fmtArgs, arrLit.Elts...)
			} else {
				fmtArgs = append(fmtArgs, args[0].Expr)
			}
			return Transform{
				Expr:    bst.Call("fmt", "Sprintf", fmtArgs...),
				Imports: []string{"fmt"},
			}
		},
	})
	// `String#+@`
	// `String#-@`
	// `String#[]`
	// `String#[]=`
	// `String#ascii_only?`
	// `String#b`
	StringType.Def("between?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(
					bst.Binary(rcvr.Expr, token.GEQ, args[0].Expr),
					token.LAND,
					bst.Binary(rcvr.Expr, token.LEQ, args[1].Expr),
				),
			}
		},
	})
	StringType.Def("bytes", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return NewArray(IntType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "StringBytes", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("bytesize", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call(nil, "len", rcvr.Expr)}
		},
	})
	// `String#byteslice`
	StringType.Def("capitalize", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Capitalize", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("capitalize!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("stdlib", "Capitalize", rcvr.Expr))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Capitalize", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("casecmp?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "EqualFold", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("center", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var pad ast.Expr
			if len(args) > 1 {
				pad = args[1].Expr
			} else {
				pad = bst.String(" ")
			}
			return Transform{
				Expr:    bst.Call("stdlib", "Center", rcvr.Expr, args[0].Expr, pad),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("chars", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return NewArray(StringType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "Split", rcvr.Expr, bst.String("")),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("chomp", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) > 0 {
				return Transform{
					Expr:    bst.Call("strings", "TrimSuffix", rcvr.Expr, args[0].Expr),
					Imports: []string{"strings"},
				}
			}
			return Transform{
				Expr: bst.Call("strings", "TrimRight", rcvr.Expr, &ast.BasicLit{
					Kind:  token.STRING,
					Value: `"` + `\r\n` + `"`,
				}),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("chomp!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var expr ast.Expr
			if len(args) > 0 {
				expr = bst.Call("strings", "TrimSuffix", rcvr.Expr, args[0].Expr)
			} else {
				expr = bst.Call("strings", "TrimRight", rcvr.Expr, &ast.BasicLit{Kind: token.STRING, Value: `"` + `\r\n` + `"`})
			}
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, expr)},
				Imports: []string{"strings"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) > 0 {
				return Transform{
					Expr:    bst.Call("strings", "TrimSuffix", rcvr.Expr, args[0].Expr),
					Imports: []string{"strings"},
				}
			}
			return Transform{
				Expr: bst.Call("strings", "TrimRight", rcvr.Expr, &ast.BasicLit{Kind: token.STRING, Value: `"` + `\r\n` + `"`}),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("chop", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// s[:len(s)-1] but handle \r\n specially — for simplicity, just trim last rune
			return Transform{
				Expr: &ast.SliceExpr{
					X:    rcvr.Expr,
					High: bst.Binary(bst.Call(nil, "len", rcvr.Expr), token.SUB, bst.Int(1)),
				},
			}
		},
	})
	StringType.Def("chop!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{bst.Assign(rcvr.Expr, &ast.SliceExpr{
					X:    rcvr.Expr,
					High: bst.Binary(bst.Call(nil, "len", rcvr.Expr), token.SUB, bst.Int(1)),
				})},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.SliceExpr{
					X:    rcvr.Expr,
					High: bst.Binary(bst.Call(nil, "len", rcvr.Expr), token.SUB, bst.Int(1)),
				},
			}
		},
	})
	StringType.Def("chr", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(nil, "string", &ast.IndexExpr{X: rcvr.Expr, Index: bst.Int(0)}),
			}
		},
	})
	// `String#clamp`
	StringType.Def("clear", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{bst.Assign(rcvr.Expr, bst.String(""))},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.String("")}
		},
	})
	StringType.Def("codepoints", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(IntType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("codepoints")
			r := it.New("r")
			loop := &ast.RangeStmt{
				Key:   ast.NewIdent("_"),
				Value: r,
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(result, bst.Call(nil, "append", result, bst.Call(nil, "int", r))),
					},
				},
			}
			return Transform{
				Stmts: []ast.Stmt{emptySlice(result, "int"), loop},
				Expr:  result,
			}
		},
	})
	StringType.Def("count", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "Count", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	// `String#crypt`
	StringType.Def("delete", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "StringDelete", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("delete!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("stdlib", "StringDelete", rcvr.Expr, args[0].Expr))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "StringDelete", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("delete_prefix", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "TrimPrefix", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("delete_prefix!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "TrimPrefix", rcvr.Expr, args[0].Expr))},
				Imports: []string{"strings"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call("strings", "TrimPrefix", rcvr.Expr, args[0].Expr), Imports: []string{"strings"}}
		},
	})
	StringType.Def("delete_suffix", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "TrimSuffix", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("delete_suffix!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "TrimSuffix", rcvr.Expr, args[0].Expr))},
				Imports: []string{"strings"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call("strings", "TrimSuffix", rcvr.Expr, args[0].Expr), Imports: []string{"strings"}}
		},
	})
	StringType.Def("downcase", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "ToLower", rcvr.Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("downcase!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "ToLower", rcvr.Expr))},
				Imports: []string{"strings"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call("strings", "ToLower", rcvr.Expr), Imports: []string{"strings"}}
		},
	})
	// `String#dump`
	// `String#each_byte`
	StringType.Def("each_char", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{StringType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			blankUnusedBlockArgs(blk)

			// for _, ch := range str { body using string(ch) }
			runeVar := it.New("ch")
			// Replace uses of block arg with string(runeVar)
			charConvert := bst.Define(blk.Args[0], bst.Call(nil, "string", runeVar))
			blk.Statements = append([]ast.Stmt{charConvert}, blk.Statements...)

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: runeVar,
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})
	// `String#each_codepoint`
	// `String#each_grapheme_cluster`
	StringType.Def("each_line", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{StringType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			blankUnusedBlockArgs(blk)

			lines := it.New("lines")
			splitCall := bst.Call("strings", "Split", rcvr.Expr, &ast.BasicLit{Kind: token.STRING, Value: `"\n"`})
			linesSplit := bst.Define(lines, splitCall)

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     lines,
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:    rcvr.Expr,
				Stmts:   []ast.Stmt{linesSplit, loop},
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("empty?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.EQL, bst.String("")),
			}
		},
	})
	StringType.Def("encode", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: rcvr.Expr} // Go strings are UTF-8; encode is a no-op
		},
	})
	// `String#encode!`
	// `String#encoding`
	StringType.Def("end_with?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "HasSuffix", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	// `String#force_encoding`
	StringType.Def("freeze", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: rcvr.Expr}
		},
	})
	// `String#getbyte`
	// `String#grapheme_clusters`
	StringType.Def("gsub", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{StringType}
		},
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if blk != nil {
				// gsub with block: regex.ReplaceAllStringFunc(s, func(match string) string { ... })
				funcLit := &ast.FuncLit{
					Type: &ast.FuncType{
						Params: &ast.FieldList{
							List: []*ast.Field{{
								Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
								Type:  ast.NewIdent("string"),
							}},
						},
						Results: &ast.FieldList{
							List: []*ast.Field{{Type: ast.NewIdent("string")}},
						},
					},
					Body: &ast.BlockStmt{List: blk.Statements},
				}
				subVar := it.New("subbed")
				stmt := bst.Define(subVar, bst.Call(args[0].Expr, "ReplaceAllStringFunc", rcvr.Expr, funcLit))
				return Transform{
					Expr:  subVar,
					Stmts: []ast.Stmt{stmt},
				}
			}
			if len(args) > 1 {
				if _, ok := args[1].Type.(Hash); ok {
					// gsub with hash: regex.ReplaceAllStringFunc(s, func(match) { h[match] || match })
					matchParam := it.New("match")
					v := it.New("v")
					okIdent := it.New("ok")
					funcLit := &ast.FuncLit{
						Type: &ast.FuncType{
							Params: &ast.FieldList{
								List: []*ast.Field{{
									Names: []*ast.Ident{matchParam},
									Type:  ast.NewIdent("string"),
								}},
							},
							Results: &ast.FieldList{
								List: []*ast.Field{{Type: ast.NewIdent("string")}},
							},
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.IfStmt{
									Init: &ast.AssignStmt{
										Lhs: []ast.Expr{v, okIdent},
										Tok: token.DEFINE,
										Rhs: []ast.Expr{&ast.IndexExpr{X: bst.Dot(args[1].Expr, "Data"), Index: matchParam}},
									},
									Cond: okIdent,
									Body: &ast.BlockStmt{
										List: []ast.Stmt{
											&ast.ReturnStmt{Results: []ast.Expr{v}},
										},
									},
								},
								&ast.ReturnStmt{Results: []ast.Expr{matchParam}},
							},
						},
					}
					subVar := it.New("subbed")
					stmt := bst.Define(subVar, bst.Call(args[0].Expr, "ReplaceAllStringFunc", rcvr.Expr, funcLit))
					return Transform{
						Expr:  subVar,
						Stmts: []ast.Stmt{stmt},
					}
				}
			}
			// String args: use strings.ReplaceAll
			if args[0].Type == StringType {
				return Transform{
					Expr:    bst.Call("strings", "ReplaceAll", rcvr.Expr, args[0].Expr, args[1].Expr),
					Imports: []string{"strings"},
				}
			}
			// Regex args: use regexp ReplaceAllString
			sub := bst.Call("stdlib", "ConvertFromGsub", UnwrapTypeExprs(args)...)
			subVar := it.New("subbed")
			stmt := bst.Define(subVar, bst.Call(args[0].Expr, "ReplaceAllString", rcvr.Expr, sub))
			return Transform{
				Expr:    subVar,
				Stmts:   []ast.Stmt{stmt},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("gsub!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{StringType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if blk != nil {
				funcLit := &ast.FuncLit{
					Type: &ast.FuncType{
						Params: &ast.FieldList{
							List: []*ast.Field{{
								Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
								Type:  ast.NewIdent("string"),
							}},
						},
						Results: &ast.FieldList{
							List: []*ast.Field{{Type: ast.NewIdent("string")}},
						},
					},
					Body: &ast.BlockStmt{List: blk.Statements},
				}
				return Transform{
					Stmts: []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call(args[0].Expr, "ReplaceAllStringFunc", rcvr.Expr, funcLit))},
				}
			}
			if args[0].Type == StringType {
				return Transform{
					Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "ReplaceAll", rcvr.Expr, args[0].Expr, args[1].Expr))},
					Imports: []string{"strings"},
				}
			}
			sub := bst.Call("stdlib", "ConvertFromGsub", UnwrapTypeExprs(args)...)
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call(args[0].Expr, "ReplaceAllString", rcvr.Expr, sub))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if blk != nil {
				funcLit := &ast.FuncLit{
					Type: &ast.FuncType{
						Params: &ast.FieldList{
							List: []*ast.Field{{
								Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
								Type:  ast.NewIdent("string"),
							}},
						},
						Results: &ast.FieldList{
							List: []*ast.Field{{Type: ast.NewIdent("string")}},
						},
					},
					Body: &ast.BlockStmt{List: blk.Statements},
				}
				subVar := it.New("subbed")
				return Transform{
					Expr:  subVar,
					Stmts: []ast.Stmt{bst.Define(subVar, bst.Call(args[0].Expr, "ReplaceAllStringFunc", rcvr.Expr, funcLit))},
				}
			}
			if args[0].Type == StringType {
				return Transform{
					Expr:    bst.Call("strings", "ReplaceAll", rcvr.Expr, args[0].Expr, args[1].Expr),
					Imports: []string{"strings"},
				}
			}
			sub := bst.Call("stdlib", "ConvertFromGsub", UnwrapTypeExprs(args)...)
			subVar := it.New("subbed")
			return Transform{
				Expr:    subVar,
				Stmts:   []ast.Stmt{bst.Define(subVar, bst.Call(args[0].Expr, "ReplaceAllString", rcvr.Expr, sub))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("hex", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Hex", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("include?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "Contains", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("index", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "Index", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("insert", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// s.insert(idx, str) → s[:idx] + str + s[idx:]
			idx := args[0].Expr
			str := args[1].Expr
			return Transform{
				Expr: bst.Binary(
					bst.Binary(
						&ast.SliceExpr{X: rcvr.Expr, High: idx},
						token.ADD,
						str,
					),
					token.ADD,
					&ast.SliceExpr{X: rcvr.Expr, Low: idx},
				),
			}
		},
	})
	// `String#intern`
	// `String#length`
	StringType.Def("lines", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(StringType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "Split", rcvr.Expr, &ast.BasicLit{Kind: token.STRING, Value: `"\n"`}),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("ljust", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var pad ast.Expr
			if len(args) > 1 {
				pad = args[1].Expr
			} else {
				pad = bst.String(" ")
			}
			return Transform{
				Expr:    bst.Call("stdlib", "Ljust", rcvr.Expr, args[0].Expr, pad),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("lstrip", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "TrimLeft", rcvr.Expr, bst.String(`\x00\t\n\v\f\r `)),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("lstrip!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "TrimLeft", rcvr.Expr, bst.String(`\x00\t\n\v\f\r `)))},
				Imports: []string{"strings"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call("strings", "TrimLeft", rcvr.Expr, bst.String(`\x00\t\n\v\f\r `)), Imports: []string{"strings"}}
		},
	})
	StringType.Def("match", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return MatchDataType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "NewMatchData", args[0].Expr, rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("match?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(args[0].Expr, "MatchString", rcvr.Expr),
			}
		},
	})
	StringType.Def("succ", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "StringSucc", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Alias("succ", "next")
	StringType.Def("oct", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Oct", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("ord", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(nil, "int", &ast.IndexExpr{
					X:     bst.Call(nil, "[]rune", rcvr.Expr),
					Index: bst.Int(0),
				}),
			}
		},
	})
	StringType.Def("partition", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(StringType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Partition", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("prepend", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{bst.Assign(rcvr.Expr, bst.Binary(args[0].Expr, token.ADD, rcvr.Expr))},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(args[0].Expr, token.ADD, rcvr.Expr),
			}
		},
	})
	StringType.Def("replace", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{bst.Assign(rcvr.Expr, args[0].Expr)},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: args[0].Expr}
		},
	})
	StringType.Def("reverse", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Reverse", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos"},
			}
		},
	})
	StringType.Alias("reverse", "reverse!")
	StringType.Def("rindex", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "LastIndex", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("rjust", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var pad ast.Expr
			if len(args) > 1 {
				pad = args[1].Expr
			} else {
				pad = bst.String(" ")
			}
			return Transform{
				Expr:    bst.Call("stdlib", "Rjust", rcvr.Expr, args[0].Expr, pad),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("rpartition", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(StringType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Rpartition", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("rstrip", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "TrimRight", rcvr.Expr, bst.String(`\x00\t\n\v\f\r `)),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("rstrip!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "TrimRight", rcvr.Expr, bst.String(`\x00\t\n\v\f\r `)))},
				Imports: []string{"strings"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call("strings", "TrimRight", rcvr.Expr, bst.String(`\x00\t\n\v\f\r `)), Imports: []string{"strings"}}
		},
	})
	StringType.Def("scan", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return NewArray(StringType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// pattern.FindAllString(str, -1)
			return Transform{
				Expr: bst.Call(args[0].Expr, "FindAllString", rcvr.Expr, &ast.UnaryExpr{Op: token.SUB, X: bst.Int(1)}),
			}
		},
	})
	// `String#scrub`
	// `String#scrub!`
	// `String#setbyte`
	StringType.Def("size", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(nil, "len", rcvr.Expr),
			}
		},
	})
	StringType.Def("length", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(nil, "len", rcvr.Expr),
			}
		},
	})
	// `String#slice`
	// `String#slice!`
	StringType.Def("split", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return NewArray(StringType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) == 0 {
				return Transform{
					Expr:    bst.Call("strings", "Fields", rcvr.Expr),
					Imports: []string{"strings"},
				}
			}
			switch args[0].Type {
			case RegexpType:
				callArgs := []ast.Expr{rcvr.Expr}
				if len(args) == 2 {
					callArgs = append(callArgs, args[1].Expr)
				} else {
					callArgs = append(callArgs, bst.Int(-1))
				}
				return Transform{
					Expr:    bst.Call(args[0].Expr, "Split", callArgs...),
					Imports: []string{"regexp"},
				}
			case StringType:
				var expr ast.Expr
				if len(args) == 1 {
					expr = bst.Call("strings", "Split", rcvr.Expr, args[0].Expr)
				} else {
					expr = bst.Call("strings", "SplitN", rcvr.Expr, args[0].Expr, args[1].Expr)
				}
				return Transform{
					Expr:    expr,
					Imports: []string{"strings"},
				}
			}
			return Transform{}
		},
	})
	StringType.Def("squeeze", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			callArgs := []ast.Expr{rcvr.Expr}
			if len(args) > 0 {
				callArgs = append(callArgs, args[0].Expr)
			}
			return Transform{
				Expr:    bst.Call("stdlib", "Squeeze", callArgs...),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("squeeze!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			callArgs := []ast.Expr{rcvr.Expr}
			if len(args) > 0 {
				callArgs = append(callArgs, args[0].Expr)
			}
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("stdlib", "Squeeze", callArgs...))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			callArgs := []ast.Expr{rcvr.Expr}
			if len(args) > 0 {
				callArgs = append(callArgs, args[0].Expr)
			}
			return Transform{
				Expr:    bst.Call("stdlib", "Squeeze", callArgs...),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("start_with?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "HasPrefix", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("strip", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "TrimSpace", rcvr.Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("strip!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "TrimSpace", rcvr.Expr))},
				Imports: []string{"strings"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call("strings", "TrimSpace", rcvr.Expr), Imports: []string{"strings"}}
		},
	})
	StringType.Def("sub", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if args[0].Type == StringType {
				return Transform{
					Expr:    bst.Call("strings", "Replace", rcvr.Expr, args[0].Expr, args[1].Expr, bst.Int(1)),
					Imports: []string{"strings"},
				}
			}
			// Regex args: use stdlib.Sub for replace-first semantics
			return Transform{
				Expr:    bst.Call("stdlib", "Sub", rcvr.Expr, args[0].Expr, args[1].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("sub!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if args[0].Type == StringType {
				return Transform{
					Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "Replace", rcvr.Expr, args[0].Expr, args[1].Expr, bst.Int(1)))},
					Imports: []string{"strings"},
				}
			}
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("stdlib", "Sub", rcvr.Expr, args[0].Expr, args[1].Expr))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if args[0].Type == StringType {
				return Transform{
					Expr:    bst.Call("strings", "Replace", rcvr.Expr, args[0].Expr, args[1].Expr, bst.Int(1)),
					Imports: []string{"strings"},
				}
			}
			return Transform{
				Expr:    bst.Call("stdlib", "Sub", rcvr.Expr, args[0].Expr, args[1].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `String#succ!`
	// `String#sum`
	StringType.Def("swapcase", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Swapcase", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	StringType.Def("swapcase!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("stdlib", "Swapcase", rcvr.Expr))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call("stdlib", "Swapcase", rcvr.Expr), Imports: []string{"github.com/redneckbeard/thanos/stdlib"}}
		},
	})
	// `String#to_c`
	StringType.Def("to_f", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			asFloat, underscore := it.New("asFloat"), it.Get("_")
			parseFloat := bst.Define([]ast.Expr{asFloat, underscore}, bst.Call("strconv", "ParseFloat", rcvr.Expr, bst.Int(64)))
			return Transform{
				Stmts:   []ast.Stmt{parseFloat},
				Expr:    asFloat,
				Imports: []string{"strconv"},
			}
		},
	})
	StringType.Def("to_i", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			asInt, underscore := it.New("asInt"), it.Get("_")
			atoi := bst.Define([]ast.Expr{asInt, underscore}, bst.Call("strconv", "Atoi", rcvr.Expr))
			return Transform{
				Stmts:   []ast.Stmt{atoi},
				Expr:    asInt,
				Imports: []string{"strconv"},
			}
		},
	})
	// `String#to_r`
	// `String#to_str`
	// `String#to_sym`
	StringType.Def("tr", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Tr", rcvr.Expr, args[0].Expr, args[1].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `String#tr!`
	// `String#tr_s`
	// `String#tr_s!`
	// `String#undump`
	// `String#unicode_normalize`
	// `String#unicode_normalize!`
	// `String#unicode_normalized?`
	// `String#unpack`
	// `String#unpack1`
	StringType.Def("upcase", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "ToUpper", rcvr.Expr),
				Imports: []string{"strings"},
			}
		},
	})
	StringType.Def("upcase!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) { return StringType, nil },
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("strings", "ToUpper", rcvr.Expr))},
				Imports: []string{"strings"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call("strings", "ToUpper", rcvr.Expr), Imports: []string{"strings"}}
		},
	})
	StringType.Def("upto", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{StringType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			s := blk.Args[0].(*ast.Ident)
			// for s := start; s <= stop; s = stdlib.StringSucc(s) { ... }
			loop := &ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: []ast.Expr{s},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{rcvr.Expr},
				},
				Cond: bst.Binary(s, token.LEQ, args[0].Expr),
				Post: bst.Assign(s, bst.Call("stdlib", "StringSucc", s)),
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			stripBlockReturn(blk)
			return Transform{
				Stmts:   []ast.Stmt{loop},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `String#valid_encoding?`

	StringType.Def("freeze", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// No-op in Go — all strings are immutable
			return Transform{Expr: rcvr.Expr}
		},
	})

	// encoding — always returns "UTF-8" (Go strings are UTF-8)
	StringType.Def("encoding", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.String("UTF-8")}
		},
	})

	// encode — no-op (Go strings are already UTF-8)
	StringType.Def("encode", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: rcvr.Expr}
		},
	})

	StringType.Def("bytes", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(IntType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "StringBytes", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	StringType.Def("<<", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{bst.OpAssign("+")(rcvr.Expr, args[0].Expr)},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.ADD, args[0].Expr),
			}
		},
	})
	StringType.Def("concat", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{bst.OpAssign("+")(rcvr.Expr, args[0].Expr)},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.ADD, args[0].Expr),
			}
		},
	})

}

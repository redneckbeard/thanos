package types

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Hash struct {
	Key, Value Type
	Instance   instance
	HasDefault bool
}

var HashClass = NewClass("Hash", "Object", nil, ClassRegistry)

func NewHash(k, v Type) Type {
	return Hash{Key: k, Value: v, Instance: HashClass.Instance}
}

func NewDefaultHash(k, v Type) Type {
	return Hash{Key: k, Value: v, Instance: HashClass.Instance, HasDefault: true}
}

func (t Hash) Equals(t2 Type) bool { return t == t2 }
func (t Hash) String() string      { return fmt.Sprintf("Hash(%s:%s)", t.Key, t.Value) }
func (t Hash) GoType() string {
	if t.HasDefault {
		return fmt.Sprintf("*stdlib.DefaultHash[%s, %s]", t.Key.GoType(), t.Value.GoType())
	}
	return fmt.Sprintf("*stdlib.OrderedMap[%s, %s]", t.Key.GoType(), t.Value.GoType())
}
func (t Hash) IsComposite() bool   { return true }
func (t Hash) Outer() Type         { return Hash{} }
func (t Hash) Inner() Type         { return t.Value }
func (t Hash) ClassName() string   { return "Hash" }
func (t Hash) IsMultiple() bool    { return false }

func (t Hash) HasMethod(method string) bool {
	return t.Instance.HasMethod(method)
}

func (t Hash) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.Instance.MustResolve(m).ReturnType(t, b, args)
}

func (t Hash) BlockArgTypes(m string, args []Type) []Type {
	return t.Instance.MustResolve(m).blockArgs(t, args)
}

func (t Hash) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	if t.HasDefault {
		switch m {
		case "[]":
			return Transform{Expr: bst.Call(rcvr, "Get", args[0].Expr)}
		case "[]=":
			setCall := bst.Call(rcvr, "Set", args[0].Expr, args[1].Expr)
			return Transform{
				Expr:  args[1].Expr,
				Stmts: []ast.Stmt{&ast.ExprStmt{X: setCall}},
			}
		default:
			// DefaultHash embeds *OrderedMap, so all methods are promoted
			transform := t.Instance.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
			return transform
		}
	}
	return t.Instance.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Hash) GetMethodSpec(m string) (MethodSpec, bool) {
	return t.Instance.Resolve(m)
}

func init() {
	// `Hash#<`
	// `Hash#<=`
	// `Hash#>`
	// `Hash#>=`
	HashClass.Instance.Def("[]", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Hash).Value, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// Generate: hash.Data[key]
			indexExpr := &ast.IndexExpr{
				X:     bst.Dot(rcvr.Expr, "Data"),
				Index: args[0].Expr,
			}
			return Transform{Expr: indexExpr}
		},
	})
	HashClass.Instance.Def("[]=", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return args[1], nil // Returns the assigned value
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// Generate: hash.Set(key, value)
			setCall := bst.Call(rcvr.Expr, "Set", args[0].Expr, args[1].Expr)
			return Transform{
				Expr:  args[1].Expr, // Return the assigned value
				Stmts: []ast.Stmt{&ast.ExprStmt{X: setCall}},
			}
		},
	})
	HashClass.Instance.Def("all?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("allTrue")
			resultDecl := bst.Define(result, it.Get("true"))

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: finalStatement.Results[0]},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(result, it.Get("false")),
						&ast.BranchStmt{Tok: token.BREAK},
					},
				},
			}

			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{resultDecl, loop},
			}
		},
	})
	HashClass.Instance.Def("any?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("anyTrue")
			resultDecl := &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{result},
							Type:  it.Get("bool"),
						},
					},
				},
			}

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(result, it.Get("true")),
						&ast.BranchStmt{Tok: token.BREAK},
					},
				},
			}

			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{resultDecl, loop},
			}
		},
	})
	// `Hash#assoc`
	// `Hash#chain`
	// `Hash#chunk`
	// `Hash#chunk_while`
	HashClass.Instance.Def("clear", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{&ast.ExprStmt{X: bst.Call(rcvr.Expr, "Clear")}},
			}
		},
	})
	HashClass.Instance.Alias("map", "collect")
	// `Hash#collect_concat`
	// `Hash#compact`
	// `Hash#compact!`
	// `Hash#compare_by_identity`
	// `Hash#compare_by_identity?`
	HashClass.Instance.Def("count", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if blk == nil {
				return Transform{Expr: bst.Call(rcvr.Expr, "Len")}
			}
			countVar := it.New("count")
			countDecl := &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{countVar},
							Type:  it.Get("int"),
						},
					},
				},
			}

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{&ast.IncDecStmt{X: countVar, Tok: token.INC}},
				},
			}

			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  countVar,
				Stmts: []ast.Stmt{countDecl, loop},
			}
		},
	})
	// `Hash#cycle`
	// `Hash#deconstruct_keys`
	// `Hash#default`
	// `Hash#default=`
	// `Hash#default_proc`
	// `Hash#default_proc=`
	HashClass.Instance.Def("delete", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Hash).Value, nil
		},
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{
					&ast.ExprStmt{X: bst.Call(rcvr.Expr, "Delete", args[0].Expr)},
				},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			val := it.New("val")
			decl := &ast.DeclStmt{
				Decl: bst.Declare(token.VAR, val, it.Get(rcvr.Type.(Hash).Value.GoType())),
			}
			cond := &ast.IfStmt{
				Init: bst.Define(
					[]ast.Expr{it.Get("v"), it.Get("ok")},
					&ast.IndexExpr{
						X:     bst.Dot(rcvr.Expr, "Data"),
						Index: args[0].Expr,
					}),
				Cond: it.Get("ok"),
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(val, it.Get("v")),
						&ast.ExprStmt{X: bst.Call(rcvr.Expr, "Delete", args[0].Expr)},
					},
				},
			}
			if blk != nil {
				finalIdx := len(blk.Statements) - 1
				final := blk.Statements[finalIdx].(*ast.ReturnStmt)
				blk.Statements[finalIdx] = bst.Assign(val, final.Results[0])
				blk.Statements = append([]ast.Stmt{bst.Define(blk.Args[0], args[0].Expr)}, blk.Statements...)
				cond.Else = &ast.BlockStmt{
					List: blk.Statements,
				}
			}

			return Transform{
				Expr:  val,
				Stmts: []ast.Stmt{decl, cond},
			}
		},
	})
	HashClass.Instance.Def("delete_if", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key, r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
			}
			final := blk.Statements[len(blk.Statements)-1]
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: final.(*ast.ReturnStmt).Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{X: bst.Call(rcvr.Expr, "Delete", blk.Args[0])},
					},
				},
			}
			loop.Body = &ast.BlockStmt{
				List: blk.Statements,
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})
	// `Hash#detect`
	// `Hash#dig`
	// `Hash#drop`
	// `Hash#drop_while`
	HashClass.Instance.Def("each", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key, r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			blankUnusedBlockArgs(blk)

			// If the receiver was remapped (e.g. by transform_values!), propagate
			// the new key/value types to the block params so that deferred interp
			// verb resolution picks up the correct types.
			if ident, ok := rcvr.Expr.(*ast.Ident); ok {
				if vt := it.GoType(ident.Name + ".value"); vt != "" {
					if vi, ok := blk.Args[1].(*ast.Ident); ok {
						it.SetType(vi.Name, vt)
					}
				}
				if kt := it.GoType(ident.Name + ".key"); kt != "" {
					if ki, ok := blk.Args[0].(*ast.Ident); ok {
						it.SetType(ki.Name, kt)
					}
				}
			}

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})
	// `Hash#each_cons`
	// `Hash#each_entry`
	HashClass.Instance.Def("each_key", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: it.Get("_"),
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})
	// `Hash#each_pair`
	// `Hash#each_slice`
	HashClass.Instance.Def("each_value", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})
	HashClass.Instance.Def("each_with_index", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h, IntType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// blk.Args layout (after destructuring): [key, value, index]
			idx := blk.Args[2]
			idxDecl := &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{idx.(*ast.Ident)},
							Type:  it.Get("int"),
						},
					},
				},
			}

			stripBlockReturn(blk)
			blankUnusedBlockArgs(blk)
			blk.Statements = append(blk.Statements, &ast.IncDecStmt{X: idx, Tok: token.INC})

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{idxDecl, loop},
			}
		},
	})
	HashClass.Instance.Def("each_with_object", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h, args[0]}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return args[0], nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// blk.Args layout (after destructuring): [key, value, accumulator]
			acc := blk.Args[2]
			accInit := bst.Define(acc, args[0].Expr)

			stripBlockReturn(blk)
			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:  acc,
				Stmts: []ast.Stmt{accInit, loop},
			}
		},
	})
	HashClass.Instance.Def("empty?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			call := bst.Binary(bst.Call(rcvr.Expr, "Len"), token.EQL, bst.Int(0))
			return Transform{Expr: call}
		},
	})
	// `Hash#entries`
	// `Hash#fetch`
	// `Hash#fetch_values`
	HashClass.Instance.Def("filter", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key, r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			filtered := it.New("filtered")
			h := rcvr.Type.(Hash)
			decl := bst.Define(filtered, bst.Call("stdlib", fmt.Sprintf("NewOrderedMap[%s, %s]", h.Key.GoType(), h.Value.GoType())))
			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
			}
			final := blk.Statements[len(blk.Statements)-1]
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: final.(*ast.ReturnStmt).Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{X: bst.Call(filtered, "Set", blk.Args[0], blk.Args[1])},
					},
				},
			}
			loop.Body = &ast.BlockStmt{
				List: blk.Statements,
			}

			return Transform{
				Expr:    filtered,
				Stmts:   []ast.Stmt{decl, loop},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `Hash#filter!`
	// `Hash#filter_map`
	// `Hash#find`
	HashClass.Instance.Alias("filter", "find_all")
	// `Hash#find_index`
	HashClass.Instance.Def("fetch", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Hash).Value, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			dataExpr := bst.Dot(rcvr.Expr, "Data")
			val := it.New("val")
			ok := it.New("ok")
			if len(args) >= 2 {
				// fetch(key, default) → val, ok := h.Data[key]; if !ok { val = default }
				return Transform{
					Stmts: []ast.Stmt{
						bst.Define(
							[]ast.Expr{val, ok},
							[]ast.Expr{&ast.IndexExpr{X: dataExpr, Index: args[0].Expr}},
						),
						&ast.IfStmt{
							Cond: &ast.UnaryExpr{Op: token.NOT, X: ok},
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									bst.Assign(val, args[1].Expr),
								},
							},
						},
					},
					Expr: val,
				}
			}
			// fetch(key) → h.Data[key]
			return Transform{
				Expr: &ast.IndexExpr{X: dataExpr, Index: args[0].Expr},
			}
		},
	})
	// `Hash#first`
	HashClass.Instance.Def("flat_map", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if arr, ok := b.(Array); ok {
				return arr, nil
			}
			return NewArray(b), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var elemType string
			if arr, ok := blk.ReturnType.(Array); ok {
				elemType = arr.Element.GoType()
			} else {
				elemType = blk.ReturnType.GoType()
			}
			result := it.New("result")
			resultInit := emptySlice(result, elemType)

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			appendCall := &ast.CallExpr{
				Fun:      it.Get("append"),
				Args:     []ast.Expr{result, finalStatement.Results[0]},
				Ellipsis: 1, // variadic spread: append(result, items...)
			}
			blk.Statements[len(blk.Statements)-1] = bst.Assign(result, appendCall)

			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{resultInit, loop},
			}
		},
	})
	// `Hash#flatten`
	// `Hash#grep`
	// `Hash#grep_v`
	// `Hash#group_by`
	HashClass.Instance.Def("has_key?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "HasKey", args[0].Expr),
			}
		},
	})
	HashClass.Instance.Def("has_value?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "HasValue", args[0].Expr),
			}
		},
	})
	HashClass.Instance.Alias("has_key?", "include?")
	// `Hash#index`
	// `Hash#inject`
	// `Hash#invert`
	HashClass.Instance.Alias("select!", "keep_if")
	// `Hash#key`
	// `Hash#key?`
	// `Hash#keys`
	HashClass.Instance.Def("keys", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r.(Hash).Key), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			keys := it.New("keys")

			return Transform{
				Stmts: []ast.Stmt{bst.Define(keys, bst.Call(rcvr.Expr, "Keys"))},
				Expr:  keys,
			}
		},
	})
	// `Hash#lazy`
	// `Hash#length`
	HashClass.Instance.Def("length", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call(rcvr.Expr, "Len")}
		},
	})
	HashClass.Instance.Def("map", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(b), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			mapped := it.New("mapped")
			targetSliceVarInit := emptySlice(mapped, blk.ReturnType.GoType())

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			transformedFinal := bst.Assign(
				mapped,
				bst.Call(nil, "append", mapped, finalStatement.Results[0]),
			)
			blk.Statements[len(blk.Statements)-1] = transformedFinal

			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:  mapped,
				Stmts: []ast.Stmt{targetSliceVarInit, loop},
			}
		},
	})
	// `Hash#max`
	// `Hash#max_by`
	HashClass.Instance.Alias("include?", "member?")
	HashClass.Instance.Def("merge", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			merged := it.New("merged")
			if blk != nil {
				h := rcvr.Type.(Hash)
				funcLit := &ast.FuncLit{
					Type: &ast.FuncType{
						Params: &ast.FieldList{
							List: []*ast.Field{
								{Names: []*ast.Ident{blk.Args[0].(*ast.Ident)}, Type: ast.NewIdent(h.Key.GoType())},
								{Names: []*ast.Ident{blk.Args[1].(*ast.Ident)}, Type: ast.NewIdent(h.Value.GoType())},
								{Names: []*ast.Ident{blk.Args[2].(*ast.Ident)}, Type: ast.NewIdent(h.Value.GoType())},
							},
						},
						Results: &ast.FieldList{
							List: []*ast.Field{{Type: ast.NewIdent(h.Value.GoType())}},
						},
					},
					Body: &ast.BlockStmt{List: blk.Statements},
				}
				return Transform{
					Stmts:   []ast.Stmt{bst.Define(merged, bst.Call("stdlib", "OrderedMapMergeBlock", rcvr.Expr, args[0].Expr, funcLit))},
					Expr:    merged,
					Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
				}
			}
			return Transform{
				Stmts:   []ast.Stmt{bst.Define(merged, bst.Call("stdlib", "OrderedMapMerge", rcvr.Expr, args[0].Expr))},
				Expr:    merged,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	HashClass.Instance.Def("merge!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			k := it.New("k")
			v := it.New("v")
			loop := &ast.RangeStmt{
				Key:   k,
				Value: v,
				Tok:   token.DEFINE,
				X:     bst.Call(args[0].Expr, "All"),
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{X: bst.Call(rcvr.Expr, "Set", k, v)},
					},
				},
			}
			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})
	HashClass.Instance.Alias("merge!", "update")
	// `Hash#min`
	// `Hash#min_by`
	// `Hash#minmax`
	// `Hash#minmax_by`
	HashClass.Instance.Def("none?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("noneTrue")
			resultDecl := bst.Define(result, it.Get("true"))

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(result, it.Get("false")),
						&ast.BranchStmt{Tok: token.BREAK},
					},
				},
			}

			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{resultDecl, loop},
			}
		},
	})
	// `Hash#one?`
	// `Hash#partition`
	// `Hash#rassoc`
	// `Hash#reduce`
	// `Hash#rehash`
	HashClass.Instance.Def("reject", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key, r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			rejected := it.New("rejected")
			h := rcvr.Type.(Hash)
			decl := bst.Define(rejected, bst.Call("stdlib", fmt.Sprintf("NewOrderedMap[%s, %s]", h.Key.GoType(), h.Value.GoType())))
			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
			}
			final := blk.Statements[len(blk.Statements)-1]
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: final.(*ast.ReturnStmt).Results[0]},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{X: bst.Call(rejected, "Set", blk.Args[0], blk.Args[1])},
					},
				},
			}
			loop.Body = &ast.BlockStmt{
				List: blk.Statements,
			}

			return Transform{
				Expr:    rejected,
				Stmts:   []ast.Stmt{decl, loop},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	HashClass.Instance.Def("reduce", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			// reduce(init) { |acc, (k, v)| ... } — acc first, then hash pair for destructuring
			return []Type{args[0], h}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return b, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// blk.Args layout (after destructuring): [acc, key, value]
			acc := blk.Args[0]
			accInit := bst.Define(acc, args[0].Expr)

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = bst.Assign(acc, finalStatement.Results[0])

			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[1],
				Value: blk.Args[2],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  acc,
				Stmts: []ast.Stmt{accInit, loop},
			}
		},
	})
	HashClass.Instance.Alias("reduce", "inject")
	HashClass.Instance.Def("reject!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key, r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			keysToDelete := it.New("keysToDelete")
			h := rcvr.Type.(Hash)
			keysInit := emptySlice(keysToDelete, h.Key.GoType())

			final := blk.Statements[len(blk.Statements)-1]
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: final.(*ast.ReturnStmt).Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(keysToDelete, bst.Call(nil, "append", keysToDelete, blk.Args[0])),
					},
				},
			}

			collectLoop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			delK := it.New("k")
			deleteLoop := &ast.RangeStmt{
				Key:   ast.NewIdent("_"),
				Value: delK,
				Tok:   token.DEFINE,
				X:     keysToDelete,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{X: bst.Call(rcvr.Expr, "Delete", delK)},
					},
				},
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{keysInit, collectLoop, deleteLoop},
			}
		},
	})
	// `Hash#replace`
	// `Hash#reverse_each`
	HashClass.Instance.Alias("filter", "select")
	HashClass.Instance.Alias("each", "each_pair")
	HashClass.Instance.Def("select!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key, r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			keysToDelete := it.New("keysToDelete")
			h := rcvr.Type.(Hash)
			keysInit := emptySlice(keysToDelete, h.Key.GoType())

			final := blk.Statements[len(blk.Statements)-1]
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: final.(*ast.ReturnStmt).Results[0]},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(keysToDelete, bst.Call(nil, "append", keysToDelete, blk.Args[0])),
					},
				},
			}

			collectLoop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			delK := it.New("k")
			deleteLoop := &ast.RangeStmt{
				Key:   ast.NewIdent("_"),
				Value: delK,
				Tok:   token.DEFINE,
				X:     keysToDelete,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{X: bst.Call(rcvr.Expr, "Delete", delK)},
					},
				},
			}

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{keysInit, collectLoop, deleteLoop},
			}
		},
	})
	HashClass.Instance.Alias("select!", "filter!")
	HashClass.Instance.Def("shift", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(AnyType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			k := it.New("shiftK")
			v := it.New("shiftV")
			call := bst.Call("stdlib", "OrderedMapShift", rcvr.Expr)
			assign := &ast.AssignStmt{
				Lhs: []ast.Expr{k, v},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{call},
			}
			pairExpr := &ast.CompositeLit{
				Type: ast.NewIdent("[]any"),
				Elts: []ast.Expr{k, v},
			}
			return Transform{
				Stmts:   []ast.Stmt{assign},
				Expr:    pairExpr,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	HashClass.Instance.Def("size", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{Expr: bst.Call(rcvr.Expr, "Len")}
		},
	})
	// `Hash#slice`
	// `Hash#slice_after`
	// `Hash#slice_before`
	// `Hash#slice_when`
	// `Hash#sort`
	HashClass.Instance.Def("sort_by", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			h := rcvr.Type.(Hash)
			result := it.New("sorted")

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			returnType := blk.ReturnType.GoType()

			// Build func literal: func(k KeyType, v ValueType) ReturnType { ... ; return expr }
			fnBody := make([]ast.Stmt, len(blk.Statements))
			copy(fnBody, blk.Statements)
			fnBody[len(fnBody)-1] = finalStatement

			fnLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Names: []*ast.Ident{blk.Args[0].(*ast.Ident)}, Type: ast.NewIdent(h.Key.GoType())},
							{Names: []*ast.Ident{blk.Args[1].(*ast.Ident)}, Type: ast.NewIdent(h.Value.GoType())},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: ast.NewIdent(returnType)}},
					},
				},
				Body: &ast.BlockStmt{List: fnBody},
			}

			call := bst.Call("stdlib", fmt.Sprintf("OrderedMapSortBy[%s, %s, %s]", h.Key.GoType(), h.Value.GoType(), returnType), rcvr.Expr, fnLit)

			return Transform{
				Stmts:   []ast.Stmt{bst.Define(result, call)},
				Expr:    result,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	HashClass.Instance.Def("min_by", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(AnyType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			h := rcvr.Type.(Hash)
			resultK := it.New("minK")
			resultV := it.New("minV")

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			returnType := blk.ReturnType.GoType()

			fnBody := make([]ast.Stmt, len(blk.Statements))
			copy(fnBody, blk.Statements)
			fnBody[len(fnBody)-1] = finalStatement

			fnLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Names: []*ast.Ident{blk.Args[0].(*ast.Ident)}, Type: ast.NewIdent(h.Key.GoType())},
							{Names: []*ast.Ident{blk.Args[1].(*ast.Ident)}, Type: ast.NewIdent(h.Value.GoType())},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: ast.NewIdent(returnType)}},
					},
				},
				Body: &ast.BlockStmt{List: fnBody},
			}

			call := bst.Call("stdlib", fmt.Sprintf("OrderedMapMinBy[%s, %s, %s]", h.Key.GoType(), h.Value.GoType(), returnType), rcvr.Expr, fnLit)
			assign := &ast.AssignStmt{
				Lhs: []ast.Expr{resultK, resultV},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{call},
			}

			// Return as []any{k, v} for use
			pairExpr := &ast.CompositeLit{
				Type: ast.NewIdent("[]any"),
				Elts: []ast.Expr{resultK, resultV},
			}

			return Transform{
				Stmts:   []ast.Stmt{assign},
				Expr:    pairExpr,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	HashClass.Instance.Def("max_by", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(AnyType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			h := rcvr.Type.(Hash)
			resultK := it.New("maxK")
			resultV := it.New("maxV")

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			returnType := blk.ReturnType.GoType()

			fnBody := make([]ast.Stmt, len(blk.Statements))
			copy(fnBody, blk.Statements)
			fnBody[len(fnBody)-1] = finalStatement

			fnLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Names: []*ast.Ident{blk.Args[0].(*ast.Ident)}, Type: ast.NewIdent(h.Key.GoType())},
							{Names: []*ast.Ident{blk.Args[1].(*ast.Ident)}, Type: ast.NewIdent(h.Value.GoType())},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: ast.NewIdent(returnType)}},
					},
				},
				Body: &ast.BlockStmt{List: fnBody},
			}

			call := bst.Call("stdlib", fmt.Sprintf("OrderedMapMaxBy[%s, %s, %s]", h.Key.GoType(), h.Value.GoType(), returnType), rcvr.Expr, fnLit)
			assign := &ast.AssignStmt{
				Lhs: []ast.Expr{resultK, resultV},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{call},
			}

			pairExpr := &ast.CompositeLit{
				Type: ast.NewIdent("[]any"),
				Elts: []ast.Expr{resultK, resultV},
			}

			return Transform{
				Stmts:   []ast.Stmt{assign},
				Expr:    pairExpr,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `Hash#store`
	HashClass.Instance.Def("sum", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			h := r.(Hash)
			return []Type{h.Key, h.Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if b != nil {
				return b, nil
			}
			return r.(Hash).Value, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			sumVar := it.New("sum")
			sumType := blk.ReturnType.GoType()
			sumDecl := &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{sumVar},
							Type:  it.Get(sumType),
						},
					},
				},
			}

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = bst.OpAssign("+")(sumVar, finalStatement.Results[0])

			blankUnusedBlockArgs(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  sumVar,
				Stmts: []ast.Stmt{sumDecl, loop},
			}
		},
	})
	// `Hash#take`
	// `Hash#take_while`
	// `Hash#tally`
	HashClass.Instance.Def("to_a", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(AnyType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("pairs")
			return Transform{
				Stmts:   []ast.Stmt{bst.Define(result, bst.Call("stdlib", "OrderedMapToA", rcvr.Expr))},
				Expr:    result,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `Hash#to_h`
	// `Hash#to_hash`
	// `Hash#to_proc`
	HashClass.Instance.Def("transform_keys", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Key}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewHash(b, r.(Hash).Value), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("transformed")
			h := rcvr.Type.(Hash)

			decl := bst.Define(result, bst.Call("stdlib", fmt.Sprintf("NewOrderedMap[%s, %s]", blk.ReturnType.GoType(), h.Value.GoType())))

			k := blk.Args[0]
			v := it.New("v")

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.ExprStmt{
				X: bst.Call(result, "Set", finalStatement.Results[0], v),
			}

			loop := &ast.RangeStmt{
				Key:   k,
				Value: v,
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:    result,
				Stmts:   []ast.Stmt{decl, loop},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `Hash#transform_keys!`
	HashClass.Instance.Def("transform_values", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewHash(r.(Hash).Key, b), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("transformed")
			h := rcvr.Type.(Hash)

			decl := bst.Define(result, bst.Call("stdlib", fmt.Sprintf("NewOrderedMap[%s, %s]", h.Key.GoType(), blk.ReturnType.GoType())))

			k := it.New("k")
			v := blk.Args[0]

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.ExprStmt{
				X: bst.Call(result, "Set", k, finalStatement.Results[0]),
			}

			loop := &ast.RangeStmt{
				Key:   k,
				Value: v,
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:    result,
				Stmts:   []ast.Stmt{decl, loop},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	HashClass.Instance.Def("transform_values!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Hash).Value}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewHash(r.(Hash).Key, b), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			h := rcvr.Type.(Hash)
			k := it.New("k")
			v := blk.Args[0]
			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)

			if blk.ReturnType == h.Value {
				// Same value type: modify in place
				blk.Statements[len(blk.Statements)-1] = &ast.ExprStmt{
					X: bst.Call(rcvr.Expr, "Set", k, finalStatement.Results[0]),
				}

				loop := &ast.RangeStmt{
					Key:   k,
					Value: v,
					Tok:   token.DEFINE,
					X:     bst.Call(rcvr.Expr, "All"),
					Body:  &ast.BlockStmt{List: blk.Statements},
				}

				return Transform{
					Expr:  rcvr.Expr,
					Stmts: []ast.Stmt{loop},
				}
			}

			// Different value type: build new map, remap variable
			rcvrName := rcvr.Expr.(*ast.Ident).Name
			newGoType := NewHash(h.Key, blk.ReturnType).GoType()
			remapped := it.Remap(rcvrName, newGoType)
			it.SetType(remapped.Name+".key", h.Key.GoType())
			it.SetType(remapped.Name+".value", blk.ReturnType.GoType())

			decl := bst.Define(remapped, bst.Call("stdlib", fmt.Sprintf("NewOrderedMap[%s, %s]", h.Key.GoType(), blk.ReturnType.GoType())))

			blk.Statements[len(blk.Statements)-1] = &ast.ExprStmt{
				X: bst.Call(remapped, "Set", k, finalStatement.Results[0]),
			}

			loop := &ast.RangeStmt{
				Key:   k,
				Value: v,
				Tok:   token.DEFINE,
				X:     bst.Call(rcvr.Expr, "All"),
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:    remapped,
				Stmts:   []ast.Stmt{decl, loop},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `Hash#uniq`
	// `Hash#update`
	// `Hash#value?`
	HashClass.Instance.Def("values", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r.(Hash).Value), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			vals := it.New("vals")

			return Transform{
				Stmts: []ast.Stmt{bst.Define(vals, bst.Call(rcvr.Expr, "Values"))},
				Expr:  vals,
			}
		},
	})
	// Hash#invert → stdlib.MapInvert(h)
	HashClass.Instance.Def("invert", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			h := r.(Hash)
			return NewHash(h.Value, h.Key), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("inverted")
			return Transform{
				Stmts:   []ast.Stmt{bst.Define(result, bst.Call("stdlib", "OrderedMapInvert", rcvr.Expr))},
				Expr:    result,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	// Hash#dig(key, ...) → nested access with nil checks
	// For now, support single key (same as []) and two keys (nested hash)
	HashClass.Instance.Def("dig", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			h := r.(Hash)
			// Walk through nested hash types
			t := h.Value
			for i := 1; i < len(args); i++ {
				if inner, ok := t.(Hash); ok {
					t = inner.Value
				}
			}
			return t, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) == 1 {
				return Transform{
					Expr: &ast.IndexExpr{X: bst.Dot(rcvr.Expr, "Data"), Index: args[0].Expr},
				}
			}
			// For multiple keys, chain .Data access
			var expr ast.Expr = bst.Dot(rcvr.Expr, "Data")
			for i, arg := range args {
				expr = &ast.IndexExpr{X: expr, Index: arg.Expr}
				// For nested hashes, add .Data for the next level
				if i < len(args)-1 {
					expr = bst.Dot(expr, "Data")
				}
			}
			return Transform{Expr: expr}
		},
	})

	// Hash#key?(k) — alias for has_key?
	HashClass.Instance.Def("key?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "HasKey", args[0].Expr),
			}
		},
	})

	// Hash#value?(v) — alias for has_value?
	HashClass.Instance.Def("value?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "HasValue", args[0].Expr),
			}
		},
	})

	// Hash#key(value) — reverse lookup, returns first key for value
	HashClass.Instance.Def("key", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Hash).Key, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "OrderedMapKey", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	HashClass.Instance.Def("values_at", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r.(Hash).Value), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("vals")
			h := rcvr.Type.(Hash)
			resultInit := emptySlice(result, h.Value.GoType())

			var loopStmts []ast.Stmt
			keySlice := it.New("keys")
			keyElts := make([]ast.Expr, len(args))
			for i, a := range args {
				keyElts[i] = a.Expr
			}
			keysInit := bst.Define(keySlice, &ast.CompositeLit{
				Type: ast.NewIdent("[]" + h.Key.GoType()),
				Elts: keyElts,
			})

			v := it.New("v")
			loop := &ast.RangeStmt{
				Key:   ast.NewIdent("_"),
				Value: it.New("k"),
				Tok:   token.DEFINE,
				X:     keySlice,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Define(v, &ast.IndexExpr{X: bst.Dot(rcvr.Expr, "Data"), Index: it.Get("k")}),
						bst.Assign(result, bst.Call(nil, "append", result, v)),
					},
				},
			}
			loopStmts = append(loopStmts, resultInit, keysInit, loop)

			return Transform{
				Stmts: loopStmts,
				Expr:  result,
			}
		},
	})
	// `Hash#zip`

	// Hash.new(default_value) — class method
	// TransformAST is handled directly in compiler/expr.go:CompileHashNew
	// because it needs access to the refined variable type from the scope chain.
	// TODO: Block form Hash.new { |h, k| ... } requires circular type inference
	// between block args and block body. Deferred for now.
	HashClass.Def("new", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if len(args) > 0 {
				return NewDefaultHash(AnyType, args[0]), nil
			}
			if b != nil {
				return NewDefaultHash(AnyType, b), nil
			}
			return NewHash(AnyType, AnyType), nil
		},
		blockArgs: func(r Type, args []Type) []Type {
			// Hash.new { |h, k| ... } — h is the hash, k is the key
			return []Type{NewHash(AnyType, AnyType), AnyType}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			panic("Hash.new TransformAST should be handled by CompileHashNew in the compiler")
		},
	})
}

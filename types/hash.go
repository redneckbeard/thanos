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
}

var HashClass = NewClass("Hash", "Object", nil, ClassRegistry)

func NewHash(k, v Type) Hash {
	return Hash{Key: k, Value: v, Instance: HashClass.Instance}
}

func (t Hash) Equals(t2 Type) bool             { return t == t2 }
func (t Hash) String() string                  { return fmt.Sprintf("Hash(%s:%s)", t.Key, t.Value) }
func (t Hash) GoType() string                  { return fmt.Sprintf("map[%s]%s", t.Key.GoType(), t.Value.GoType()) }
func (t Hash) IsComposite() bool               { return true }
func (t Hash) Outer() Type                     { return Hash{} }
func (t Hash) ClassName() string               { return "Hash" }
func (t Hash) IsMultiple() bool                { return false }
func (t Hash) SupportsBrackets(t1 Type) string { return "" }

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
	return t.Instance.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func init() {
	// `Hash#<`
	// `Hash#<=`
	// `Hash#>`
	// `Hash#>=`
	// `Hash#[]`
	// `Hash#[]=`
	// `Hash#all?`
	// `Hash#any?`
	// `Hash#assoc`
	// `Hash#chain`
	// `Hash#chunk`
	// `Hash#chunk_while`
	// `Hash#clear`
	// `Hash#collect`
	// `Hash#collect_concat`
	// `Hash#compact`
	// `Hash#compact!`
	// `Hash#compare_by_identity`
	// `Hash#compare_by_identity?`
	// `Hash#count`
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
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			val := it.New("val")
			decl := &ast.DeclStmt{
				Decl: bst.Declare(token.VAR, val, it.Get(rcvr.Type.(Hash).Value.GoType())),
			}
			cond := &ast.IfStmt{
				Init: bst.Define(
					[]ast.Expr{it.Get("v"), it.Get("ok")},
					&ast.IndexExpr{
						X:     rcvr.Expr,
						Index: args[0].Expr,
					}),
				Cond: it.Get("ok"),
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(val, it.Get("v")),
						&ast.ExprStmt{X: bst.Call(nil, "delete", rcvr.Expr, args[0].Expr)},
					},
				},
			}
			if blk != nil {
				//TODO build utility for ReturnStatement unwrapping
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
				X:     rcvr.Expr,
			}
			final := blk.Statements[len(blk.Statements)-1]
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: final.(*ast.ReturnStmt).Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{X: bst.Call(nil, "delete", rcvr.Expr, blk.Args[0])},
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
			//TODO duplicative, see Array#each
			var transformedFinal *ast.ExprStmt
			finalStatement := blk.Statements[len(blk.Statements)-1]
			switch f := finalStatement.(type) {
			case *ast.ReturnStmt:
				transformedFinal = &ast.ExprStmt{
					X: f.Results[0],
				}
			case *ast.ExprStmt:
				transformedFinal = f
			default:
				panic("Encountered an unexpected node type")
			}

			blk.Statements[len(blk.Statements)-1] = transformedFinal

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
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
			//TODO duplicative, see Array#each
			var transformedFinal *ast.ExprStmt
			finalStatement := blk.Statements[len(blk.Statements)-1]
			switch f := finalStatement.(type) {
			case *ast.ReturnStmt:
				transformedFinal = &ast.ExprStmt{
					X: f.Results[0],
				}
			case *ast.ExprStmt:
				transformedFinal = f
			default:
				panic("Encountered an unexpected node type")
			}

			blk.Statements[len(blk.Statements)-1] = transformedFinal

			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: it.Get("_"),
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
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
			//TODO duplicative, see Array#each
			var transformedFinal *ast.ExprStmt
			finalStatement := blk.Statements[len(blk.Statements)-1]
			switch f := finalStatement.(type) {
			case *ast.ReturnStmt:
				transformedFinal = &ast.ExprStmt{
					X: f.Results[0],
				}
			case *ast.ExprStmt:
				transformedFinal = f
			default:
				panic("Encountered an unexpected node type")
			}

			blk.Statements[len(blk.Statements)-1] = transformedFinal

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
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
	// `Hash#each_with_index`
	// `Hash#each_with_object`
	HashClass.Instance.Def("empty?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			call := bst.Binary(rcvr.Expr, token.EQL, bst.Call(nil, "len", rcvr.Expr))
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
			//TODO nearly entirely the same as #delete_if
			filtered := it.New("filtered")
			decl := bst.Define(filtered, bst.Call(nil, "make", it.Get(rcvr.Type.GoType())))
			loop := &ast.RangeStmt{
				Key:   blk.Args[0],
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
			}
			final := blk.Statements[len(blk.Statements)-1]
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: final.(*ast.ReturnStmt).Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(
							&ast.IndexExpr{X: filtered, Index: blk.Args[0]},
							blk.Args[1],
						),
					},
				},
			}
			loop.Body = &ast.BlockStmt{
				List: blk.Statements,
			}

			return Transform{
				Expr:  filtered,
				Stmts: []ast.Stmt{decl, loop},
			}
		},
	})
	// `Hash#filter!`
	// `Hash#filter_map`
	// `Hash#find`
	// `Hash#find_all`
	// `Hash#find_index`
	// `Hash#first`
	// `Hash#flat_map`
	// `Hash#flatten`
	// `Hash#grep`
	// `Hash#grep_v`
	// `Hash#group_by`
	HashClass.Instance.Def("has_key?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			hasKey := it.New("hasKey")
			decl := &ast.DeclStmt{Decl: bst.Declare(token.VAR, hasKey, it.Get("bool"))}
			cond := &ast.IfStmt{
				Init: bst.Define(
					[]ast.Expr{it.Get("_"), it.Get("ok")},
					&ast.IndexExpr{
						X:     rcvr.Expr,
						Index: args[0].Expr,
					}),
				Cond: it.Get("ok"),
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(hasKey, it.Get("true")),
					},
				},
			}
			return Transform{
				Stmts: []ast.Stmt{decl, cond},
				Expr:  hasKey,
			}
		},
	})
	HashClass.Instance.Def("has_value?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "MapHasValue", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	HashClass.Instance.Def("include?", MethodSpec{})
	// `Hash#index`
	// `Hash#inject`
	// `Hash#invert`
	// `Hash#keep_if`
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
				Stmts:   []ast.Stmt{bst.Define(keys, bst.Call("stdlib", "MapKeys", rcvr.Expr))},
				Expr:    keys,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
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
			call := bst.Call(nil, "len", rcvr.Expr)
			return Transform{Expr: call}
		},
	})
	// `Hash#map`
	// `Hash#max`
	// `Hash#max_by`
	// `Hash#member?`
	HashClass.Instance.Def("merge", MethodSpec{})
	// `Hash#merge!`
	// `Hash#min`
	// `Hash#min_by`
	// `Hash#minmax`
	// `Hash#minmax_by`
	// `Hash#none?`
	// `Hash#one?`
	// `Hash#partition`
	// `Hash#rassoc`
	// `Hash#reduce`
	// `Hash#rehash`
	HashClass.Instance.Def("reject", MethodSpec{})
	// `Hash#reject!`
	// `Hash#replace`
	// `Hash#reverse_each`
	// `Hash#select`
	// `Hash#select!`
	// `Hash#shift`
	// `Hash#size`
	// `Hash#slice`
	// `Hash#slice_after`
	// `Hash#slice_before`
	// `Hash#slice_when`
	// `Hash#sort`
	// `Hash#sort_by`
	// `Hash#store`
	// `Hash#sum`
	// `Hash#take`
	// `Hash#take_while`
	// `Hash#tally`
	// `Hash#to_a`
	// `Hash#to_h`
	// `Hash#to_hash`
	// `Hash#to_proc`
	// `Hash#transform_keys`
	// `Hash#transform_keys!`
	// `Hash#transform_values`
	// `Hash#transform_values!`
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
				Stmts:   []ast.Stmt{bst.Define(vals, bst.Call("stdlib", "MapValues", rcvr.Expr))},
				Expr:    vals,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// `Hash#values_at`
	// `Hash#zip`
}

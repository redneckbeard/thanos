package types

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Array struct {
	Element  Type
	Instance instance
}

var ArrayClass = NewClass("Array", "Object", nil, ClassRegistry)

func NewArray(inner Type) Array {
	return Array{Element: inner, Instance: ArrayClass.Instance}
}

func (t Array) Equals(t2 Type) bool             { return t == t2 }
func (t Array) String() string                  { return fmt.Sprintf("Array(%s)", t.Element) }
func (t Array) GoType() string                  { return fmt.Sprintf("[]%s", t.Element.GoType()) }
func (t Array) IsComposite() bool               { return true }
func (t Array) Outer() Type                     { return Array{} }
func (t Array) ClassName() string               { return "Array" }
func (t Array) IsMultiple() bool                { return false }
func (t Array) SupportsBrackets(t1 Type) string { return "" }

func (t Array) HasMethod(method string) bool {
	return t.Instance.HasMethod(method)
}

func (t Array) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.Instance.MustResolve(m).ReturnType(t, b, args)
}

func (t Array) BlockArgTypes(m string, args []Type) []Type {
	return t.Instance.MustResolve(m).blockArgs(t, args)
}

func (t Array) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.Instance.MustResolve(m).TransformAST(TypeExpr{Expr: rcvr, Type: t}, args, blk, it)
}

func init() {
	// !
	// !=
	// !~
	// &
	// *
	// +
	ArrayClass.Instance.Def("-", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			subtracted := it.New("subtracted")
			return Transform{
				Stmts: []ast.Stmt{
					bst.Define(subtracted, bst.Call("stdlib", "SubtractSlice", rcvr.Expr, args[0].Expr)),
				},
				Expr:    subtracted,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	ArrayClass.Instance.Def("<<", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if args[0] != r.(Array).Element {
				return nil, fmt.Errorf("Tried to append %s to %s", args[0], r)
			}
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{
					bst.Assign(rcvr.Expr, bst.Call(nil, "append", rcvr.Expr, args[0].Expr)),
				},
				Expr: rcvr.Expr,
			}
		},
	})
	// <=>
	// ==
	// ===
	// =~
	// []
	// []=
	// __id__
	// __send__
	// all?
	// any?
	// append
	// assoc
	// at
	// bsearch
	// bsearch_index
	// chain
	// chunk
	// chunk_while
	// class
	// clear
	// clone
	// collect_concat
	// combination
	// compact
	// compact!
	// concat
	// count
	// cycle
	// deconstruct
	// define_singleton_method
	// delete
	// delete_at
	// delete_if
	// detect
	// difference
	// dig
	// display
	ArrayClass.Instance.Def("drop", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.SliceExpr{
					X:   rcvr.Expr,
					Low: args[0].Expr,
				},
			}
		},
	})

	// drop_while
	// dup
	ArrayClass.Instance.Def("each", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
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
	// each_cons
	// each_entry
	// each_index
	// each_slice
	// each_with_index
	// each_with_object
	// empty?
	// entries
	// enum_for
	// eql?
	// equal?
	// extend
	// fetch
	// fill
	// filter_map
	// find
	// find_all
	// find_index
	// first
	// flat_map
	// flatten
	// flatten!
	// freeze
	// frozen?
	// grep
	// grep_v
	// group_by
	// hash
	ArrayClass.Instance.Def("include?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if args[0] != r.(Array).Element {
				return nil, fmt.Errorf("Tried to search %s for %s", r, args[0])
			}
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			includes := it.New("includes")
			boolDecl := &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{includes},
							Type:  it.Get("bool"),
						},
					},
				},
			}

			x := it.New("x")
			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: x,
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.IfStmt{
							Cond: &ast.BinaryExpr{
								X:  x,
								Op: token.EQL,
								Y:  args[0].Expr,
							},
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									bst.Assign(includes, it.Get("true")),
									&ast.BranchStmt{Tok: token.BREAK},
								},
							},
						},
					},
				},
			}

			return Transform{
				Expr:  includes,
				Stmts: []ast.Stmt{boolDecl, loop},
			}
		},
	})
	// index
	// insert
	// inspect
	// instance_eval
	// instance_exec
	// instance_of?
	// instance_variable_defined?
	// instance_variable_get
	// instance_variable_set
	// instance_variables
	// intersection
	ArrayClass.Instance.Def("join", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if args[0] != StringType {
				return nil, fmt.Errorf("'join' takes a StringType argument but saw %s", args[0])
			}
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			joinCall := bst.Call("strings", "Join")

			if rcvr.Type.(Array).Element == StringType {
				joinCall.Args = []ast.Expr{rcvr.Expr, args[0].Expr}
				return Transform{
					Expr: joinCall,
				}
			}

			segments := it.New("segments")
			targetSlice := emptySlice(segments, "string")

			x := it.New("x")
			Sprinted := bst.Call("fmt", "Sprintf", bst.String("%v"), x)
			loop := appendLoop(x, segments, rcvr.Expr, segments, Sprinted)

			joinCall.Args = []ast.Expr{segments, args[0].Expr}

			return Transform{
				Expr:    joinCall,
				Stmts:   []ast.Stmt{targetSlice, loop},
				Imports: []string{"fmt", "strings"},
			}

		},
	})
	// keep_if
	// kind_of?
	// last
	// lazy
	ArrayClass.Instance.Def("length", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			call := bst.Call(nil, "len", rcvr.Expr)
			return Transform{Expr: call}
		},
	})
	// map!
	ArrayClass.Instance.Def("map", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
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
				Expr:  mapped,
				Stmts: []ast.Stmt{targetSliceVarInit, loop},
			}
		},
	})
	// max
	// max_by
	// member?
	// method
	// min
	// min_by
	// minmax
	// minmax_by
	// nil?
	// none?
	// object_id
	// one?
	// pack
	// partition
	// permutation
	// pop
	// private_methods
	// product
	// protected_methods
	// public_method
	// public_methods
	// public_send
	// push
	// rassoc
	ArrayClass.Instance.Def("reduce", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{args[0], r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return args[0], nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			acc := blk.Args[0]
			accInit := bst.Define(acc, args[0].Expr)

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			transformedFinal := bst.Assign(acc, finalStatement.Results)

			blk.Statements[len(blk.Statements)-1] = transformedFinal

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[1],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
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
	// reject
	// reject!
	// remove_instance_variable
	// repeated_combination
	// repeated_permutation
	// replace
	// respond_to?
	// reverse
	// reverse!
	// reverse_each
	// rindex
	// rotate
	// rotate!
	// sample
	// select!
	ArrayClass.Instance.Def("select", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			//TODO largely copied and pasted from map. Find an abstraction.
			selected := it.New("selected")
			targetSliceVarInit := emptySlice(selected, rcvr.Type.(Array).Element.GoType())

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			assignment := bst.Assign(selected, bst.Call(nil, "append", selected, blk.Args[0]))
			transformedFinal := &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{assignment},
				},
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
				Expr:  selected,
				Stmts: []ast.Stmt{targetSliceVarInit, loop},
			}
		},
	})
	// send
	// shift
	// shuffle
	// shuffle!
	// singleton_class
	// singleton_method
	// singleton_methods
	// slice
	// slice!
	// slice_after
	// slice_before
	// slice_when
	// sort
	//TODO block support
	ArrayClass.Instance.Def("sort!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var sortFunc string
			switch rcvr.Type.(Array).Element {
			case IntType:
				sortFunc = "Ints"
			case FloatType:
				sortFunc = "Float64s"
			case StringType:
				sortFunc = "Strings"
			case SymbolType:
				sortFunc = "Strings"
			default:
				sortFunc = "Sort"
			}

			sortCall := bst.Call("sort", sortFunc, rcvr.Expr)

			return Transform{
				Expr: rcvr.Expr,
				Stmts: []ast.Stmt{&ast.ExprStmt{
					X: sortCall,
				}},
				Imports: []string{"sort"},
			}
		},
	})
	// sort_by
	// sort_by!
	// sum
	ArrayClass.Instance.Def("take", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.SliceExpr{
					X:    rcvr.Expr,
					High: args[0].Expr,
				},
			}
		},
	})
	// take_while
	// tally
	// tap
	// then
	// to_a
	// to_ary
	// to_enum
	// to_h
	// to_s
	// transpose
	// union
	// uniq
	// uniq!
	ArrayClass.Instance.Def("unshift", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			newHead := &ast.CompositeLit{
				Type: &ast.ArrayType{
					Elt: it.Get(rcvr.Type.(Array).Element.GoType()),
				},
				Elts: UnwrapTypeExprs(args),
			}
			appendCall := bst.Call(nil, "append", newHead, rcvr.Expr)
			appendCall.Ellipsis = token.Pos(1)
			appendStmt := bst.Assign(rcvr.Expr, appendCall)

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{appendStmt},
			}
		},
	})
	ArrayClass.Instance.Def("values_at", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			vals := it.New("vals")
			targetSlice := emptySlice(vals, rcvr.Type.(Array).Element.GoType())

			x := it.New("x")
			loop := appendLoop(
				x,
				vals,
				&ast.CompositeLit{
					Type: &ast.ArrayType{
						Elt: it.Get("int"),
					},
					Elts: UnwrapTypeExprs(args),
				},
				vals,
				&ast.IndexExpr{
					X:     rcvr.Expr,
					Index: x,
				},
			)

			return Transform{
				Expr:  vals,
				Stmts: []ast.Stmt{targetSlice, loop},
			}
		},
	})
	// yield_self
	// zip
	// |

	// In most cases, the analogous operation in Go is going to return a new
	// slice, so most bang-suffixed methods are going to get aliased to their
	// not-in-place-modification counterparts.
	ArrayClass.Instance.Alias("length", "size")
	ArrayClass.Instance.Alias("map", "collect")
	//ArrayClass.Instance.Alias("map!", "collect!")
	ArrayClass.Instance.Alias("reduce", "inject")
	ArrayClass.Instance.Alias("select", "filter")
	//ArrayClass.Instance.Alias("select!", "filter!")
	ArrayClass.Instance.Alias("unshift", "prepend")
}

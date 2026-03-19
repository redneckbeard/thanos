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

func NewArray(inner Type) Type {
	return Array{Element: inner, Instance: ArrayClass.Instance}
}

func (t Array) Equals(t2 Type) bool { return t == t2 }
func (t Array) String() string      { return fmt.Sprintf("Array(%s)", t.Element) }
func (t Array) GoType() string      { return fmt.Sprintf("[]%s", t.Element.GoType()) }
func (t Array) IsComposite() bool   { return true }
func (t Array) Outer() Type         { return Array{} }
func (t Array) Inner() Type         { return t.Element }
func (t Array) ClassName() string   { return "Array" }
func (t Array) IsMultiple() bool    { return false }

func (t Array) HasMethod(method string) bool {
	return t.Instance.HasMethod(method)
}

func (t Array) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.Instance.MustResolve(m).ReturnType(t, b, args)
}

func (t Array) GetMethodSpec(m string) (MethodSpec, bool) {
	return t.Instance.Resolve(m)
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
	ArrayClass.Instance.Def("+", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			combined := it.New("combined")
			appendCall := bst.Call(nil, "append", rcvr.Expr, args[0].Expr)
			appendCall.Ellipsis = token.Pos(1)
			return Transform{
				Stmts: []ast.Stmt{
					bst.Define(combined, appendCall),
				},
				Expr: combined,
			}
		},
	})
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

	ArrayClass.Instance.Def("&", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Intersect", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("|", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Union", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("rotate", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var n ast.Expr = bst.Int(1)
			if len(args) > 0 {
				n = args[0].Expr
			}
			return Transform{
				Expr:    bst.Call("stdlib", "Rotate", rcvr.Expr, n),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	ArrayClass.Instance.Def("dup", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("slices", "Clone", rcvr.Expr),
				Imports: []string{"slices"},
			}
		},
	})

	ArrayClass.Instance.Def("<<", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			arrayType := r.(Array)
			argType := args[0]

			// If this is an empty array (AnyType), refine it based on the appended value
			if arrayType.Element == AnyType {
				// Update the array type to match the appended element
				refinedArray := NewArray(argType)
				return refinedArray, nil
			}

			// Normal type checking for non-empty arrays
			if argType != arrayType.Element {
				return nil, fmt.Errorf("Tried to append %s to %s", argType, r)
			}
			return r, nil
		},
		// Add a special refinement function that gets called during method call analysis
		RefineVariable: func(receiverName string, newType Type, scope interface{}) {
			if scopeChain, ok := scope.(interface{ RefineVariableType(string, Type) bool }); ok {
				scopeChain.RefineVariableType(receiverName, newType)
			}
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
	// ===
	// =~
	// __id__
	// __send__
	ArrayClass.Instance.Def("all?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("allTrue")
			resultInit := bst.Define(result, it.Get("true"))

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
				Expr:  result,
				Stmts: []ast.Stmt{resultInit, loop},
			}
		},
	})
	ArrayClass.Instance.Def("any?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
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
				Expr:  result,
				Stmts: []ast.Stmt{resultDecl, loop},
			}
		},
	})
	// append
	// assoc
	// at
	// bsearch
	// bsearch_index
	// chain
	// chunk
	// chunk_while
	// class
	ArrayClass.Instance.Def("clear", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{
					bst.Assign(rcvr.Expr, &ast.SliceExpr{
						X:    rcvr.Expr,
						High: bst.Int(0),
					}),
				},
				Expr: rcvr.Expr,
			}
		},
	})
	// clone
	// collect_concat
	ArrayClass.Instance.Def("combination", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r), nil // Array of Arrays
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Combination", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("compact", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			arr := r.(Array)
			if opt, ok := arr.Element.(Optional); ok {
				return NewArray(opt.Element), nil
			}
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			arr := rcvr.Type.(Array)
			if _, ok := arr.Element.(Optional); ok {
				return Transform{
					Expr:    bst.Call("stdlib", "Compact", rcvr.Expr),
					Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
				}
			}
			// If not Optional, compact is a no-op — return the same array
			return Transform{Expr: rcvr.Expr}
		},
	})
	// compact!
	ArrayClass.Instance.Def("concat", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			appendCall := bst.Call(nil, "append", rcvr.Expr, args[0].Expr)
			appendCall.Ellipsis = token.Pos(1)
			return Transform{
				Stmts: []ast.Stmt{
					bst.Assign(rcvr.Expr, appendCall),
				},
				Expr: rcvr.Expr,
			}
		},
	})
	// count
	// cycle
	// deconstruct
	// define_singleton_method
	// delete
	// delete_at
	// delete_if
	ArrayClass.Instance.Def("count", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if blk == nil {
				return Transform{Expr: bst.Call(nil, "len", rcvr.Expr)}
			}
			count := it.New("count")
			countDecl := &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{count},
							Type:  it.Get("int"),
						},
					},
				},
			}

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.IncDecStmt{X: count, Tok: token.INC},
					},
				},
			}

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
				Expr:  count,
				Stmts: []ast.Stmt{countDecl, loop},
			}
		},
	})
	ArrayClass.Instance.Def("delete", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformStmtAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Stmts: []ast.Stmt{
					bst.Assign(rcvr.Expr, bst.Call("stdlib", "DeleteSlice", rcvr.Expr, args[0].Expr)),
				},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "DeleteSlice", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("each_cons", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type
			funcLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{{
							Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
							Type:  ast.NewIdent(elem.GoType()),
						}},
					},
				},
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			return Transform{
				Stmts: []ast.Stmt{
					&ast.ExprStmt{X: bst.Call("stdlib", "EachCons", rcvr.Expr, args[0].Expr, funcLit)},
				},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
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

	ArrayClass.Instance.Def("drop_while", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array).Element
			funcLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{{
							Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
							Type:  ast.NewIdent(elem.GoType()),
						}},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: ast.NewIdent("bool")}},
					},
				},
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			return Transform{
				Expr:    bst.Call("stdlib", "DropWhile", rcvr.Expr, funcLit),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// dup
	ArrayClass.Instance.Def("each", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)

			// Propagate remapped element type to block param
			if ident, ok := rcvr.Expr.(*ast.Ident); ok {
				if et := it.GoType(ident.Name + ".elem"); et != "" {
					if vi, ok := blk.Args[0].(*ast.Ident); ok {
						it.SetType(vi.Name, et)
					}
				}
			}

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
	ArrayClass.Instance.Def("each_slice", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			blankUnusedBlockArgs(blk)
			i := it.New("i")
			size := args[0].Expr
			end := it.New("end")
			loop := &ast.ForStmt{
				Init: bst.Define(i, bst.Int(0)),
				Cond: bst.Binary(i, token.LSS, bst.Call(nil, "len", rcvr.Expr)),
				Post: &ast.AssignStmt{Lhs: []ast.Expr{i}, Tok: token.ADD_ASSIGN, Rhs: []ast.Expr{size}},
				Body: &ast.BlockStmt{
					List: append([]ast.Stmt{
						bst.Define(end, bst.Binary(i, token.ADD, size)),
						&ast.IfStmt{
							Cond: bst.Binary(end, token.GTR, bst.Call(nil, "len", rcvr.Expr)),
							Body: &ast.BlockStmt{List: []ast.Stmt{
								bst.Assign(end, bst.Call(nil, "len", rcvr.Expr)),
							}},
						},
						bst.Define(blk.Args[0], &ast.SliceExpr{X: rcvr.Expr, Low: i, High: end}),
					}, blk.Statements...),
				},
			}
			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})
	ArrayClass.Instance.Def("each_with_index", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element, IntType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)

			loop := &ast.RangeStmt{
				Key:   blk.Args[1],
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
	ArrayClass.Instance.Def("each_with_object", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element, args[0]}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return args[0], nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			obj := blk.Args[1]
			objInit := bst.Define(obj, args[0].Expr)

			stripBlockReturn(blk)

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
				Expr:  obj,
				Stmts: []ast.Stmt{objInit, loop},
			}
		},
	})
	// empty?
	// entries
	// enum_for
	// eql?
	// equal?
	// extend
	// fetch
	ArrayClass.Instance.Def("fill", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Fill", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// filter_map
	ArrayClass.Instance.Def("find", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewOptional(r.(Array).Element), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array).Element
			result := it.New("found")
			// var found *ElemType
			decl := &ast.DeclStmt{Decl: bst.Declare(token.VAR, result, ast.NewIdent(NewOptional(elem).GoType()))}

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			tmp := it.New("v")
			// if <condition> { v := elem; found = &v; break }
			body := &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Define(tmp, blk.Args[0]),
						bst.Assign(result, &ast.UnaryExpr{Op: token.AND, X: tmp}),
						&ast.BranchStmt{Tok: token.BREAK},
					},
				},
			}

			blk.Statements[len(blk.Statements)-1] = body

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
				Expr:  result,
				Stmts: []ast.Stmt{decl, loop},
			}
		},
	})
	ArrayClass.Instance.Def("detect", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewOptional(r.(Array).Element), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return ArrayClass.Instance.MustResolve("find").TransformAST(rcvr, args, blk, it)
		},
	})
	// find_all
	ArrayClass.Instance.Def("find_index", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("foundIdx")
			resultInit := bst.Define(result, &ast.UnaryExpr{Op: token.SUB, X: bst.Int(1)})

			idx := it.New("i")
			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(result, idx),
						&ast.BranchStmt{Tok: token.BREAK},
					},
				},
			}

			loop := &ast.RangeStmt{
				Key:   idx,
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{resultInit, loop},
			}
		},
	})
	ArrayClass.Instance.Def("flatten", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			inner := r.(Array).Element
			if arr, ok := inner.(Array); ok {
				return arr, nil
			}
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Flatten", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("first", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if len(args) == 0 {
				// first() with no args returns a single element (no nil support in thanos)
				return r.(Array).Element, nil
			}
			// first(n) returns an array
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) == 0 {
				// arr.first with no arguments - simple array access
				return Transform{
					Expr: &ast.IndexExpr{
						X:     rcvr.Expr,
						Index: bst.Int(0),
					},
				}
			}
			// arr.first(n) with argument
			n := args[0].Expr
			result := it.New("firstN")
			lenVar := it.New("arrLen")
			takeVar := it.New("take")

			// Calculate how many elements to take
			stmts := []ast.Stmt{
				bst.Define(lenVar, bst.Call(nil, "len", rcvr.Expr)),
				bst.Define(takeVar, n),
				&ast.IfStmt{
					Cond: bst.Binary(takeVar, token.GTR, lenVar),
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							bst.Assign(takeVar, lenVar),
						},
					},
				},
				bst.Define(result, &ast.SliceExpr{
					X:    rcvr.Expr,
					High: takeVar,
				}),
			}

			return Transform{
				Stmts: stmts,
				Expr:  result,
			}
		},
	})
	ArrayClass.Instance.Def("last", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if len(args) == 0 {
				// last() with no args returns a single element
				return r.(Array).Element, nil
			}
			// last(n) returns an array
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) == 0 {
				// arr.last with no arguments - access last element
				lenVar := it.New("arrLen")
				return Transform{
					Stmts: []ast.Stmt{
						bst.Define(lenVar, bst.Call(nil, "len", rcvr.Expr)),
					},
					Expr: &ast.IndexExpr{
						X: rcvr.Expr,
						Index: bst.Binary(lenVar, token.SUB, bst.Int(1)),
					},
				}
			}
			// arr.last(n) with argument - get last n elements
			n := args[0].Expr
			result := it.New("lastN")
			lenVar := it.New("arrLen")
			takeVar := it.New("take")
			startVar := it.New("start")

			stmts := []ast.Stmt{
				bst.Define(lenVar, bst.Call(nil, "len", rcvr.Expr)),
				bst.Define(takeVar, n),
				&ast.IfStmt{
					Cond: bst.Binary(takeVar, token.GTR, lenVar),
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							bst.Assign(takeVar, lenVar),
						},
					},
				},
				bst.Define(startVar, bst.Binary(lenVar, token.SUB, takeVar)),
				bst.Define(result, &ast.SliceExpr{
					X:   rcvr.Expr,
					Low: startVar,
				}),
			}

			return Transform{
				Stmts: stmts,
				Expr:  result,
			}
		},
	})
	ArrayClass.Instance.Def("empty?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// arr.empty? -> len(arr) == 0
			return Transform{
				Expr: bst.Binary(
					bst.Call(nil, "len", rcvr.Expr),
					token.EQL,
					bst.Int(0),
				),
			}
		},
	})
	ArrayClass.Instance.Def("push", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			arrayType := r.(Array)

			// Check that all arguments match the array element type
			for _, argType := range args {
				// If this is an empty array (AnyType), refine it based on the first argument
				if arrayType.Element == AnyType && len(args) > 0 {
					arrayType = NewArray(args[0]).(Array)
					// Update the return type to match the refined array
					r = arrayType
				}

				// Normal type checking
				if argType != arrayType.Element {
					return nil, fmt.Errorf("Tried to push %s to %s", argType, r)
				}
			}
			return r, nil
		},
		// Add refinement function like << operator
		RefineVariable: func(receiverName string, newType Type, scope interface{}) {
			if scopeChain, ok := scope.(interface{ RefineVariableType(string, Type) bool }); ok {
				scopeChain.RefineVariableType(receiverName, newType)
			}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stmts := []ast.Stmt{}

			// For each argument, append it to the array
			for _, arg := range args {
				stmts = append(stmts,
					bst.Assign(rcvr.Expr, bst.Call(nil, "append", rcvr.Expr, arg.Expr)),
				)
			}

			return Transform{
				Stmts: stmts,
				Expr:  rcvr.Expr,
			}
		},
	})
	ArrayClass.Instance.Def("pop", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if len(args) == 0 {
				// pop() with no args returns a single element
				return r.(Array).Element, nil
			}
			// pop(n) with argument returns an array
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) == 0 {
				// arr.pop with no arguments - remove and return last element
				lenVar := it.New("arrLen")
				result := it.New("popped")

				stmts := []ast.Stmt{
					bst.Define(lenVar, bst.Call(nil, "len", rcvr.Expr)),
					bst.Define(result, &ast.IndexExpr{
						X: rcvr.Expr,
						Index: bst.Binary(lenVar, token.SUB, bst.Int(1)),
					}),
					bst.Assign(rcvr.Expr, &ast.SliceExpr{
						X:    rcvr.Expr,
						High: bst.Binary(lenVar, token.SUB, bst.Int(1)),
					}),
				}

				return Transform{
					Stmts: stmts,
					Expr:  result,
				}
			}
			// arr.pop(n) with argument - remove and return last n elements
			n := args[0].Expr
			result := it.New("poppedN")
			lenVar := it.New("arrLen")
			takeVar := it.New("take")
			startVar := it.New("start")

			stmts := []ast.Stmt{
				bst.Define(lenVar, bst.Call(nil, "len", rcvr.Expr)),
				bst.Define(takeVar, n),
				&ast.IfStmt{
					Cond: bst.Binary(takeVar, token.GTR, lenVar),
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							bst.Assign(takeVar, lenVar),
						},
					},
				},
				bst.Define(startVar, bst.Binary(lenVar, token.SUB, takeVar)),
				// Extract the elements to return
				bst.Define(result, &ast.SliceExpr{
					X:   rcvr.Expr,
					Low: startVar,
				}),
				// Modify the original array
				bst.Assign(rcvr.Expr, &ast.SliceExpr{
					X:    rcvr.Expr,
					High: startVar,
				}),
			}

			return Transform{
				Stmts: stmts,
				Expr:  result,
			}
		},
	})
	// first
	ArrayClass.Instance.Def("flat_map", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			// block returns an array, flat_map returns the inner type as an array
			if arr, ok := b.(Array); ok {
				return arr, nil
			}
			return NewArray(b), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			mapped := it.New("flatMapped")
			targetSliceVarInit := emptySlice(mapped, blk.ReturnType.(Array).Element.GoType())

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			appendCall := bst.Call(nil, "append", mapped, finalStatement.Results[0])
			appendCall.Ellipsis = token.Pos(1)
			transformedFinal := bst.Assign(mapped, appendCall)

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
	// flatten
	// flatten!
	// freeze
	// frozen?
	// grep
	// grep_v
	ArrayClass.Instance.Def("group_by", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewHash(b, r), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array)
			keyType := blk.ReturnType
			grouped := it.New("grouped")

			// grouped := stdlib.NewOrderedMap[K, []V]()
			initMap := bst.Define(grouped, bst.Call("stdlib", fmt.Sprintf("NewOrderedMap[%s, %s]", keyType.GoType(), elem.GoType())))

			// Extract key expression from block's return statement
			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			key := it.New("key")
			keyAssign := bst.Define(key, finalStatement.Results[0])

			// grouped.Set(key, append(grouped.Data[key], x))
			dataIndex := &ast.IndexExpr{X: bst.Dot(grouped, "Data"), Index: key}
			appendStmt := &ast.ExprStmt{X: bst.Call(grouped, "Set", key, bst.Call(nil, "append", dataIndex, blk.Args[0]))}

			blk.Statements[len(blk.Statements)-1] = keyAssign

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body: &ast.BlockStmt{
					List: append(blk.Statements, appendStmt),
				},
			}

			return Transform{
				Expr:    grouped,
				Stmts:   []ast.Stmt{initMap, loop},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
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
	ArrayClass.Instance.Def("index", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("idx")
			resultInit := bst.Define(result, &ast.UnaryExpr{Op: token.SUB, X: bst.Int(1)})

			i := it.New("i")
			x := it.New("x")
			loop := &ast.RangeStmt{
				Key:   i,
				Value: x,
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.IfStmt{
							Cond: bst.Binary(x, token.EQL, args[0].Expr),
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									bst.Assign(result, i),
									&ast.BranchStmt{Tok: token.BREAK},
								},
							},
						},
					},
				},
			}

			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{resultInit, loop},
			}
		},
	})
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
			if len(args) > 0 && args[0] != StringType {
				return nil, fmt.Errorf("'join' takes a StringType argument but saw %s", args[0])
			}
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			joinCall := bst.Call("strings", "Join")

			var delimExpr ast.Expr
			if len(args) > 0 {
				delimExpr = args[0].Expr
			} else {
				delimExpr = bst.String("")
			}

			if rcvr.Type.(Array).Element == StringType {
				joinCall.Args = []ast.Expr{rcvr.Expr, delimExpr}
				return Transform{
					Expr: joinCall,
				}
			}

			segments := it.New("segments")
			targetSlice := emptySlice(segments, "string")

			x := it.New("x")
			Sprinted := bst.Call("fmt", "Sprintf", bst.String("%v"), x)
			loop := appendLoop(x, segments, rcvr.Expr, segments, Sprinted)

			joinCall.Args = []ast.Expr{segments, delimExpr}

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

			rewriteReturnsToAppend(blk.Statements, mapped)

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
	ArrayClass.Instance.Def("max", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Max", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("max_by", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array).Element
			keyType := blk.ReturnType
			funcLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{{
							Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
							Type:  ast.NewIdent(elem.GoType()),
						}},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{
							Type: ast.NewIdent(keyType.GoType()),
						}},
					},
				},
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			return Transform{
				Expr:    bst.Call("stdlib", "MaxBy", rcvr.Expr, funcLit),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// member?
	ArrayClass.Instance.Def("min", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Min", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("min_by", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array).Element
			keyType := blk.ReturnType
			funcLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{{
							Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
							Type:  ast.NewIdent(elem.GoType()),
						}},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{
							Type: ast.NewIdent(keyType.GoType()),
						}},
					},
				},
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			return Transform{
				Expr:    bst.Call("stdlib", "MinBy", rcvr.Expr, funcLit),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("none?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			result := it.New("noneTrue")
			resultInit := bst.Define(result, it.Get("true"))

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
				Expr:  result,
				Stmts: []ast.Stmt{resultInit, loop},
			}
		},
	})
	// object_id
	ArrayClass.Instance.Def("one?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			count := it.New("oneCount")
			countDecl := &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{count},
							Type:  it.Get("int"),
						},
					},
				},
			}

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.IncDecStmt{X: count, Tok: token.INC},
					},
				},
			}

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
				Expr:  bst.Binary(count, token.EQL, bst.Int(1)),
				Stmts: []ast.Stmt{countDecl, loop},
			}
		},
	})
	// pack
	ArrayClass.Instance.Def("partition", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elemGoType := rcvr.Type.(Array).Element.GoType()
			trueArr := it.New("trueArr")
			falseArr := it.New("falseArr")
			initTrue := emptySlice(trueArr, elemGoType)
			initFalse := emptySlice(falseArr, elemGoType)

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			appendTrue := bst.Assign(trueArr, bst.Call(nil, "append", trueArr, blk.Args[0]))
			appendFalse := bst.Assign(falseArr, bst.Call(nil, "append", falseArr, blk.Args[0]))
			ifStmt := &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{List: []ast.Stmt{appendTrue}},
				Else: &ast.BlockStmt{List: []ast.Stmt{appendFalse}},
			}

			blk.Statements[len(blk.Statements)-1] = ifStmt

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body: &ast.BlockStmt{
					List: blk.Statements,
				},
			}

			result := it.New("partitioned")
			sliceType := rcvr.Type.GoType()
			initResult := bst.Define(result, &ast.CompositeLit{
				Type: ast.NewIdent("[]" + sliceType),
				Elts: []ast.Expr{trueArr, falseArr},
			})

			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{initTrue, initFalse, loop, initResult},
			}
		},
	})
	ArrayClass.Instance.Def("permutation", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Permutation", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// pop
	// private_methods
	ArrayClass.Instance.Def("product", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Product", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
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
	ArrayClass.Instance.Def("reverse_each", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			blockVar := blk.Args[0]
			idx := it.New("i")
			loop := &ast.ForStmt{
				Init: bst.Define(idx, bst.Binary(bst.Call(nil, "len", rcvr.Expr), token.SUB, bst.Int(1))),
				Cond: bst.Binary(idx, token.GEQ, bst.Int(0)),
				Post: &ast.IncDecStmt{X: idx, Tok: token.DEC},
				Body: &ast.BlockStmt{
					List: append([]ast.Stmt{
						bst.Define(blockVar, &ast.IndexExpr{X: rcvr.Expr, Index: idx}),
					}, blk.Statements...),
				},
			}
			return Transform{
				Stmts: []ast.Stmt{loop},
				Expr:  rcvr.Expr,
			}
		},
	})
	ArrayClass.Instance.Def("rindex", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Rindex", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// rotate
	ArrayClass.Instance.Def("reverse", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "ReverseSlice", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
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
	ArrayClass.Instance.Def("reject", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			rejected := it.New("rejected")
			targetSliceVarInit := emptySlice(rejected, rcvr.Type.(Array).Element.GoType())

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			assignment := bst.Assign(rejected, bst.Call(nil, "append", rejected, blk.Args[0]))
			transformedFinal := &ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: finalStatement.Results[0]},
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
				Expr:  rejected,
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
	ArrayClass.Instance.Def("sort", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "SortSlice", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("sort_by", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array).Element
			keyType := blk.ReturnType
			funcLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{{
							Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
							Type:  ast.NewIdent(elem.GoType()),
						}},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{
							Type: ast.NewIdent(keyType.GoType()),
						}},
					},
				},
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			return Transform{
				Expr:    bst.Call("stdlib", "SortBy", rcvr.Expr, funcLit),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// sort_by!
	ArrayClass.Instance.Def("sum", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Sum", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
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
	ArrayClass.Instance.Def("take_while", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array).Element
			funcLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{{
							Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
							Type:  ast.NewIdent(elem.GoType()),
						}},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: ast.NewIdent("bool")}},
					},
				},
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			return Transform{
				Expr:    bst.Call("stdlib", "TakeWhile", rcvr.Expr, funcLit),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// tally
	// tap
	// then
	// to_a
	// to_ary
	// to_enum
	// to_h
	// to_s
	ArrayClass.Instance.Def("transpose", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil // Array of Arrays → Array of Arrays
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Transpose", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// union
	ArrayClass.Instance.Def("uniq", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Uniq", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	// uniq!
	ArrayClass.Instance.Def("unshift", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			arrayType := r.(Array)
			if arrayType.Element == AnyType && len(args) > 0 {
				return NewArray(args[0]), nil
			}
			return r, nil
		},
		RefineVariable: func(receiverName string, newType Type, scope interface{}) {
			if scopeChain, ok := scope.(interface{ RefineVariableType(string, Type) bool }); ok {
				scopeChain.RefineVariableType(receiverName, newType)
			}
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
	ArrayClass.Instance.Def("zip", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(NewArray(r.(Array).Element)), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Zip", rcvr.Expr, args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	ArrayClass.Instance.Def("delete_at", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			val := it.New("val")
			stmt := bst.Define([]ast.Expr{rcvr.Expr, val}, bst.Call("stdlib", "DeleteAtSlice", rcvr.Expr, args[0].Expr))
			return Transform{
				Stmts:   []ast.Stmt{stmt},
				Expr:    val,
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	ArrayClass.Instance.Def("sample", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Sample", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	ArrayClass.Instance.Def("shuffle", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Shuffle", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	ArrayClass.Instance.Def("tally", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewHash(r.(Array).Element, IntType), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "Tally", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	ArrayClass.Instance.Def("dig", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			expr := rcvr.Expr
			for _, arg := range args {
				expr = &ast.IndexExpr{X: expr, Index: arg.Expr}
			}
			return Transform{Expr: expr}
		},
	})

	ArrayClass.Instance.Def("fetch", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			idx := args[0].Expr
			if len(args) == 1 {
				return Transform{
					Expr: &ast.IndexExpr{X: rcvr.Expr, Index: idx},
				}
			}
			return Transform{
				Expr:    bst.Call("stdlib", "FetchSlice", rcvr.Expr, idx, args[1].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	ArrayClass.Instance.Def("each_index", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{IntType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			blockVar := blk.Args[0]
			loop := &ast.ForStmt{
				Init: bst.Define(blockVar, bst.Int(0)),
				Cond: bst.Binary(blockVar, token.LSS, bst.Call(nil, "len", rcvr.Expr)),
				Post: &ast.IncDecStmt{X: blockVar, Tok: token.INC},
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})

	ArrayClass.Instance.Def("map!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(b), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array).Element
			if blk.ReturnType == elem {
				// Same type: modify in place
				idx := it.New("i")
				finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
				transformedFinal := bst.Assign(
					&ast.IndexExpr{X: rcvr.Expr, Index: idx},
					finalStatement.Results[0],
				)
				blk.Statements[len(blk.Statements)-1] = transformedFinal

				loop := &ast.RangeStmt{
					Key:   idx,
					Value: blk.Args[0],
					Tok:   token.DEFINE,
					X:     rcvr.Expr,
					Body:  &ast.BlockStmt{List: blk.Statements},
				}

				return Transform{
					Expr:  rcvr.Expr,
					Stmts: []ast.Stmt{loop},
				}
			}
			// Different type: allocate new slice, remap the variable name
			rcvrName := rcvr.Expr.(*ast.Ident).Name
			newGoType := NewArray(blk.ReturnType).GoType()
			remapped := it.Remap(rcvrName, newGoType)
			it.SetType(remapped.Name+".elem", blk.ReturnType.GoType())
			targetSliceVarInit := emptySlice(remapped, blk.ReturnType.GoType())

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			transformedFinal := bst.Assign(
				remapped,
				bst.Call(nil, "append", remapped, finalStatement.Results[0]),
			)
			blk.Statements[len(blk.Statements)-1] = transformedFinal

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			return Transform{
				Expr:  remapped,
				Stmts: []ast.Stmt{targetSliceVarInit, loop},
			}
		},
	})
	ArrayClass.Instance.Def("select!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			n := it.New("n")
			nInit := bst.Define(n, bst.Int(0))

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			assignAndInc := []ast.Stmt{
				bst.Assign(&ast.IndexExpr{X: rcvr.Expr, Index: n}, blk.Args[0]),
				&ast.IncDecStmt{X: n, Tok: token.INC},
			}
			transformedFinal := &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{List: assignAndInc},
			}
			blk.Statements[len(blk.Statements)-1] = transformedFinal

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			truncate := bst.Assign(rcvr.Expr, &ast.SliceExpr{X: rcvr.Expr, High: n})

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{nInit, loop, truncate},
			}
		},
	})
	ArrayClass.Instance.Def("reject!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			n := it.New("n")
			nInit := bst.Define(n, bst.Int(0))

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			assignAndInc := []ast.Stmt{
				bst.Assign(&ast.IndexExpr{X: rcvr.Expr, Index: n}, blk.Args[0]),
				&ast.IncDecStmt{X: n, Tok: token.INC},
			}
			transformedFinal := &ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: finalStatement.Results[0]},
				Body: &ast.BlockStmt{List: assignAndInc},
			}
			blk.Statements[len(blk.Statements)-1] = transformedFinal

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: blk.Args[0],
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body:  &ast.BlockStmt{List: blk.Statements},
			}

			truncate := bst.Assign(rcvr.Expr, &ast.SliceExpr{X: rcvr.Expr, High: n})

			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{nInit, loop, truncate},
			}
		},
	})
	ArrayClass.Instance.Def("reverse!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    rcvr.Expr,
				Stmts:   []ast.Stmt{&ast.ExprStmt{X: bst.Call("slices", "Reverse", rcvr.Expr)}},
				Imports: []string{"slices"},
			}
		},
	})
	ArrayClass.Instance.Def("uniq!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    rcvr.Expr,
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("stdlib", "Uniq", rcvr.Expr))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("sort_by!", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Array).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			elem := rcvr.Type.(Array).Element
			keyType := blk.ReturnType
			funcLit := &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{{
							Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
							Type:  ast.NewIdent(elem.GoType()),
						}},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{{
							Type: ast.NewIdent(keyType.GoType()),
						}},
					},
				},
				Body: &ast.BlockStmt{List: blk.Statements},
			}
			sortCall := bst.Call("stdlib", "SortByInPlace", rcvr.Expr, funcLit)
			return Transform{
				Expr:    rcvr.Expr,
				Stmts:   []ast.Stmt{&ast.ExprStmt{X: sortCall}},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("compact!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			elem := r.(Array).Element
			if opt, ok := elem.(Optional); ok {
				return NewArray(opt.Element), nil
			}
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    rcvr.Expr,
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("stdlib", "Compact", rcvr.Expr))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("flatten!", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			elem := r.(Array).Element
			if inner, ok := elem.(Array); ok {
				return inner, nil
			}
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    rcvr.Expr,
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, bst.Call("stdlib", "Flatten", rcvr.Expr))},
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	ArrayClass.Instance.Def("delete", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			n := it.New("n")
			v := it.New("v")
			nInit := bst.Define(n, bst.Int(0))

			loop := &ast.RangeStmt{
				Key:   it.Get("_"),
				Value: v,
				Tok:   token.DEFINE,
				X:     rcvr.Expr,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.IfStmt{
							Cond: bst.Binary(v, token.NEQ, args[0].Expr),
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									bst.Assign(&ast.IndexExpr{X: rcvr.Expr, Index: n}, v),
									&ast.IncDecStmt{X: n, Tok: token.INC},
								},
							},
						},
					},
				},
			}

			truncate := bst.Assign(rcvr.Expr, &ast.SliceExpr{X: rcvr.Expr, High: n})

			return Transform{
				Expr:  args[0].Expr,
				Stmts: []ast.Stmt{nInit, loop, truncate},
			}
		},
	})
	ArrayClass.Instance.Def("shift", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Array).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			shifted := it.New("shifted")
			shiftStmt := bst.Define(shifted, &ast.IndexExpr{X: rcvr.Expr, Index: bst.Int(0)})
			reassign := bst.Assign(rcvr.Expr, &ast.SliceExpr{X: rcvr.Expr, Low: bst.Int(1)})

			return Transform{
				Expr:  shifted,
				Stmts: []ast.Stmt{shiftStmt, reassign},
			}
		},
	})
	ArrayClass.Instance.Def("insert", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// arr.insert(idx, val) => arr = slices.Insert(arr, idx, val)
			insertArgs := []ast.Expr{rcvr.Expr}
			for _, a := range args {
				insertArgs = append(insertArgs, a.Expr)
			}
			insertCall := &ast.CallExpr{
				Fun:  bst.Dot(ast.NewIdent("slices"), "Insert"),
				Args: insertArgs,
			}
			return Transform{
				Expr:    rcvr.Expr,
				Stmts:   []ast.Stmt{bst.Assign(rcvr.Expr, insertCall)},
				Imports: []string{"slices"},
			}
		},
	})

	ArrayClass.Instance.Alias("length", "size")
	ArrayClass.Instance.Alias("map", "collect")
	ArrayClass.Instance.Alias("map!", "collect!")
	ArrayClass.Instance.Alias("reduce", "inject")
	ArrayClass.Instance.Alias("select", "filter")
	ArrayClass.Instance.Alias("select!", "filter!")
	ArrayClass.Instance.Alias("unshift", "prepend")
	ArrayClass.Instance.Alias("shuffle", "shuffle!")
	ArrayClass.Instance.Alias("reject!", "delete_if")
	ArrayClass.Instance.Alias("include?", "member?")
}

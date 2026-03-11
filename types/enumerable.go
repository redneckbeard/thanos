package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

func init() {
	RegisterMixin(&Mixin{
		Name:            "Enumerable",
		RequiredMethods: []string{"each"},
		Apply: func(instance Instance, ctx MixinContext) {
			// getElementType resolves the each method's block param type lazily.
			// It returns AnyType if the type can't be determined yet.
			getElementType := func() Type {
				if eachMethod, ok := ctx["eachMethod"]; ok {
					// eachMethod is a *parser.Method but we receive it as interface{}
					// Use the BlockArgs accessor on the MethodSpec instead
					if spec, ok := instance.Resolve("each"); ok {
						args := spec.BlockArgs(instance, nil)
						if len(args) > 0 && args[0] != nil {
							return args[0]
						}
					}
					_ = eachMethod
				}
				return AnyType
			}

			// enumCallback generates: rcvr.Each(func(elem ElemType) { body })
			enumCallback := func(rcvr TypeExpr, blk *Block, it bst.IdentTracker) *ast.ExprStmt {
				elemType := getElementType()
				return &ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
						Args: []ast.Expr{
							&ast.FuncLit{
								Type: &ast.FuncType{
									Params: &ast.FieldList{
										List: []*ast.Field{{
											Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
											Type:  ast.NewIdent(elemType.GoType()),
										}},
									},
								},
								Body: &ast.BlockStmt{List: blk.Statements},
							},
						},
					},
				}
			}

			// map / collect
			instance.Def("map", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return NewArray(b), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					mapped := it.New("mapped")
					targetSliceVarInit := emptySlice(mapped, blk.ReturnType.GoType())

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = bst.Assign(
						mapped,
						bst.Call(nil, "append", mapped, finalStatement.Results[0]),
					)

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  mapped,
						Stmts: []ast.Stmt{targetSliceVarInit, loop},
					}
				},
			})
			instance.Alias("map", "collect")

			// select / filter
			instance.Def("select", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return NewArray(getElementType()), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					selected := it.New("selected")
					targetSliceVarInit := emptySlice(selected, elemType.GoType())

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
						Cond: finalStatement.Results[0],
						Body: &ast.BlockStmt{
							List: []ast.Stmt{bst.Assign(selected, bst.Call(nil, "append", selected, blk.Args[0]))},
						},
					}

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  selected,
						Stmts: []ast.Stmt{targetSliceVarInit, loop},
					}
				},
			})
			instance.Alias("select", "filter")

			// reject
			instance.Def("reject", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return NewArray(getElementType()), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					rejected := it.New("rejected")
					targetSliceVarInit := emptySlice(rejected, elemType.GoType())

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
						Cond: &ast.UnaryExpr{Op: token.NOT, X: finalStatement.Results[0]},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{bst.Assign(rejected, bst.Call(nil, "append", rejected, blk.Args[0]))},
						},
					}

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  rejected,
						Stmts: []ast.Stmt{targetSliceVarInit, loop},
					}
				},
			})

			// find / detect
			instance.Def("find", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return getElementType(), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					found := it.New("found")
					done := it.New("done")

					// var found ElemType
					foundDecl := &ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{found},
									Type:  ast.NewIdent(elemType.GoType()),
								},
							},
						},
					}
					// done := false
					doneDecl := bst.Define(done, ast.NewIdent("false"))

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
						Cond: finalStatement.Results[0],
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								bst.Assign(found, blk.Args[0]),
								bst.Assign(done, ast.NewIdent("true")),
							},
						},
					}

					// Prepend early return check
					earlyReturn := &ast.IfStmt{
						Cond: done,
						Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}},
					}
					blk.Statements = append([]ast.Stmt{earlyReturn}, blk.Statements...)

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  found,
						Stmts: []ast.Stmt{foundDecl, doneDecl, loop},
					}
				},
			})
			instance.Alias("find", "detect")

			// any?
			instance.Def("any?", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
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
									Type:  ast.NewIdent("bool"),
								},
							},
						},
					}

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
						Cond: finalStatement.Results[0],
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								bst.Assign(result, ast.NewIdent("true")),
								&ast.ReturnStmt{},
							},
						},
					}

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  result,
						Stmts: []ast.Stmt{resultDecl, loop},
					}
				},
			})

			// all?
			instance.Def("all?", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return BoolType, nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					result := it.New("allTrue")
					resultDecl := bst.Define(result, ast.NewIdent("true"))

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
						Cond: &ast.UnaryExpr{Op: token.NOT, X: finalStatement.Results[0]},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								bst.Assign(result, ast.NewIdent("false")),
								&ast.ReturnStmt{},
							},
						},
					}

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  result,
						Stmts: []ast.Stmt{resultDecl, loop},
					}
				},
			})

			// none?
			instance.Def("none?", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return BoolType, nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					result := it.New("noneTrue")
					resultDecl := bst.Define(result, ast.NewIdent("true"))

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = &ast.IfStmt{
						Cond: finalStatement.Results[0],
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								bst.Assign(result, ast.NewIdent("false")),
								&ast.ReturnStmt{},
							},
						},
					}

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  result,
						Stmts: []ast.Stmt{resultDecl, loop},
					}
				},
			})

			// count (with block)
			instance.Def("count", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return IntType, nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					count := it.New("count")
					countDecl := &ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{count},
									Type:  ast.NewIdent("int"),
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

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  count,
						Stmts: []ast.Stmt{countDecl, loop},
					}
				},
			})

			// include? (no block)
			instance.Def("include?", MethodSpec{
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return BoolType, nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					result := it.New("includes")
					resultDecl := &ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{result},
									Type:  ast.NewIdent("bool"),
								},
							},
						},
					}

					elem := it.New("elem")
					callbackBody := []ast.Stmt{
						&ast.IfStmt{
							Cond: bst.Binary(elem, token.EQL, args[0].Expr),
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									bst.Assign(result, ast.NewIdent("true")),
									&ast.ReturnStmt{},
								},
							},
						},
					}

					callback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{
											List: []*ast.Field{{
												Names: []*ast.Ident{elem},
												Type:  ast.NewIdent(elemType.GoType()),
											}},
										},
									},
									Body: &ast.BlockStmt{List: callbackBody},
								},
							},
						},
					}

					return Transform{
						Expr:  result,
						Stmts: []ast.Stmt{resultDecl, callback},
					}
				},
			})

			// each_with_index
			instance.Def("each_with_index", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType(), IntType}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return r, nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					stripBlockReturn(blk)
					blankUnusedBlockArgs(blk)

					idx := it.New("idx")
					idxDecl := &ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{idx},
									Type:  ast.NewIdent("int"),
								},
							},
						},
					}

					// Append idx++ at end of callback body, and inject idx as second block arg
					blk.Statements = append(blk.Statements, &ast.IncDecStmt{X: idx, Tok: token.INC})

					callbackParams := []*ast.Field{{
						Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
						Type:  ast.NewIdent(elemType.GoType()),
					}}

					// Map second block arg to idx
					if len(blk.Args) > 1 {
						// Replace uses of the second block arg with idx
						replaceIdent(blk.Statements, blk.Args[1].(*ast.Ident), idx)
					}

					callback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{List: callbackParams},
									},
									Body: &ast.BlockStmt{List: blk.Statements},
								},
							},
						},
					}

					return Transform{
						Stmts: []ast.Stmt{idxDecl, callback},
					}
				},
			})

			// reduce / inject
			instance.Def("reduce", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					elemType := getElementType()
					return []Type{elemType, elemType}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					if len(args) > 0 {
						return args[0], nil
					}
					return getElementType(), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					acc := it.New("acc")
					first := it.New("first")

					var stmts []ast.Stmt

					// If initial value provided: acc := initialValue
					// Otherwise: var acc ElemType; first := true (use first element as initial)
					if len(args) > 0 {
						stmts = append(stmts, bst.Define(acc, args[0].Expr))
					} else {
						stmts = append(stmts,
							&ast.DeclStmt{
								Decl: &ast.GenDecl{
									Tok: token.VAR,
									Specs: []ast.Spec{
										&ast.ValueSpec{
											Names: []*ast.Ident{acc},
											Type:  ast.NewIdent(elemType.GoType()),
										},
									},
								},
							},
							bst.Define(first, ast.NewIdent("true")),
						)
					}

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = bst.Assign(acc, finalStatement.Results[0])

					// Replace block's accumulator arg with our acc variable
					replaceIdent(blk.Statements, blk.Args[0].(*ast.Ident), acc)

					var callbackBody []ast.Stmt
					if len(args) == 0 {
						// first iteration: acc = elem, skip block
						callbackBody = append(callbackBody, &ast.IfStmt{
							Cond: first,
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									bst.Assign(acc, blk.Args[1].(*ast.Ident)),
									bst.Assign(first, ast.NewIdent("false")),
									&ast.ReturnStmt{},
								},
							},
						})
					}
					callbackBody = append(callbackBody, blk.Statements...)

					callback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{
											List: []*ast.Field{{
												Names: []*ast.Ident{blk.Args[1].(*ast.Ident)},
												Type:  ast.NewIdent(elemType.GoType()),
											}},
										},
									},
									Body: &ast.BlockStmt{List: callbackBody},
								},
							},
						},
					}

					stmts = append(stmts, callback)

					return Transform{
						Expr:  acc,
						Stmts: stmts,
					}
				},
			})
			instance.Alias("reduce", "inject")

			// sum (no block)
			instance.Def("sum", MethodSpec{
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return getElementType(), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					sum := it.New("sum")
					sumDecl := &ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{sum},
									Type:  ast.NewIdent(elemType.GoType()),
								},
							},
						},
					}

					elem := it.New("elem")
					callbackBody := []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{sum},
							Tok: token.ADD_ASSIGN,
							Rhs: []ast.Expr{elem},
						},
					}

					callback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{
											List: []*ast.Field{{
												Names: []*ast.Ident{elem},
												Type:  ast.NewIdent(elemType.GoType()),
											}},
										},
									},
									Body: &ast.BlockStmt{List: callbackBody},
								},
							},
						},
					}

					return Transform{
						Expr:  sum,
						Stmts: []ast.Stmt{sumDecl, callback},
					}
				},
			})

			// min (no block)
			instance.Def("min", MethodSpec{
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return getElementType(), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					result := it.New("minVal")
					first := it.New("first")

					resultDecl := &ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{result},
									Type:  ast.NewIdent(elemType.GoType()),
								},
							},
						},
					}
					firstDecl := bst.Define(first, ast.NewIdent("true"))

					elem := it.New("elem")
					callbackBody := []ast.Stmt{
						&ast.IfStmt{
							Cond: bst.Binary(first, token.LOR, bst.Binary(elem, token.LSS, result)),
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									bst.Assign(result, elem),
									bst.Assign(first, ast.NewIdent("false")),
								},
							},
						},
					}

					callback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{
											List: []*ast.Field{{
												Names: []*ast.Ident{elem},
												Type:  ast.NewIdent(elemType.GoType()),
											}},
										},
									},
									Body: &ast.BlockStmt{List: callbackBody},
								},
							},
						},
					}

					return Transform{
						Expr:  result,
						Stmts: []ast.Stmt{resultDecl, firstDecl, callback},
					}
				},
			})

			// max (no block)
			instance.Def("max", MethodSpec{
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return getElementType(), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					result := it.New("maxVal")
					first := it.New("first")

					resultDecl := &ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{result},
									Type:  ast.NewIdent(elemType.GoType()),
								},
							},
						},
					}
					firstDecl := bst.Define(first, ast.NewIdent("true"))

					elem := it.New("elem")
					callbackBody := []ast.Stmt{
						&ast.IfStmt{
							Cond: bst.Binary(first, token.LOR, bst.Binary(elem, token.GTR, result)),
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									bst.Assign(result, elem),
									bst.Assign(first, ast.NewIdent("false")),
								},
							},
						},
					}

					callback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{
											List: []*ast.Field{{
												Names: []*ast.Ident{elem},
												Type:  ast.NewIdent(elemType.GoType()),
											}},
										},
									},
									Body: &ast.BlockStmt{List: callbackBody},
								},
							},
						},
					}

					return Transform{
						Expr:  result,
						Stmts: []ast.Stmt{resultDecl, firstDecl, callback},
					}
				},
			})

			// sort — collect into slice, then sort
			instance.Def("sort", MethodSpec{
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return NewArray(getElementType()), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					sorted := it.New("sorted")
					sliceInit := emptySlice(sorted, elemType.GoType())

					elem := it.New("elem")
					collectBody := []ast.Stmt{
						bst.Assign(sorted, bst.Call(nil, "append", sorted, elem)),
					}

					collectCallback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{
											List: []*ast.Field{{
												Names: []*ast.Ident{elem},
												Type:  ast.NewIdent(elemType.GoType()),
											}},
										},
									},
									Body: &ast.BlockStmt{List: collectBody},
								},
							},
						},
					}

					// sort.Slice or slices.Sort
					sortCall := &ast.ExprStmt{
						X: bst.Call("stdlib", "SortSlice", sorted),
					}

					return Transform{
						Expr:    sorted,
						Stmts:   []ast.Stmt{sliceInit, collectCallback, sortCall},
						Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
					}
				},
			})

			// sort_by
			instance.Def("sort_by", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return NewArray(getElementType()), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					sorted := it.New("sorted")
					sliceInit := emptySlice(sorted, elemType.GoType())

					// Collect all elements into a slice
					collectElem := it.New("elem")
					collectBody := []ast.Stmt{
						bst.Assign(sorted, bst.Call(nil, "append", sorted, collectElem)),
					}

					collectCallback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{
											List: []*ast.Field{{
												Names: []*ast.Ident{collectElem},
												Type:  ast.NewIdent(elemType.GoType()),
											}},
										},
									},
									Body: &ast.BlockStmt{List: collectBody},
								},
							},
						},
					}

					// Sort using the block as key function
					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					sortKeyExpr := finalStatement.Results[0]
					blk.Statements = blk.Statements[:len(blk.Statements)-1]

					sortCall := &ast.ExprStmt{
						X: bst.Call("stdlib", "SortByInPlace", sorted,
							&ast.FuncLit{
								Type: &ast.FuncType{
									Params: &ast.FieldList{
										List: []*ast.Field{{
											Names: []*ast.Ident{blk.Args[0].(*ast.Ident)},
											Type:  ast.NewIdent(elemType.GoType()),
										}},
									},
									Results: &ast.FieldList{
										List: []*ast.Field{{
											Type: ast.NewIdent(blk.ReturnType.GoType()),
										}},
									},
								},
								Body: &ast.BlockStmt{
									List: append(blk.Statements, &ast.ReturnStmt{Results: []ast.Expr{sortKeyExpr}}),
								},
							},
						),
					}

					return Transform{
						Expr:    sorted,
						Stmts:   []ast.Stmt{sliceInit, collectCallback, sortCall},
						Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
					}
				},
			})

			// flat_map
			instance.Def("flat_map", MethodSpec{
				blockArgs: func(r Type, args []Type) []Type {
					return []Type{getElementType()}
				},
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					if arr, ok := b.(Array); ok {
						return arr, nil
					}
					return NewArray(b), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					result := it.New("flatMapped")
					var innerType string
					if arr, ok := blk.ReturnType.(Array); ok {
						innerType = arr.Element.GoType()
					} else {
						innerType = blk.ReturnType.GoType()
					}
					resultInit := emptySlice(result, innerType)

					finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
					blk.Statements[len(blk.Statements)-1] = bst.Assign(
						result,
						bst.Call(nil, "append", result, &ast.CallExpr{
							Fun:      finalStatement.Results[0],
							Ellipsis: 1,
						}),
					)

					loop := enumCallback(rcvr, blk, it)
					return Transform{
						Expr:  result,
						Stmts: []ast.Stmt{resultInit, loop},
					}
				},
			})

			// to_a — collect into array
			instance.Def("to_a", MethodSpec{
				ReturnType: func(r Type, b Type, args []Type) (Type, error) {
					return NewArray(getElementType()), nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					elemType := getElementType()
					result := it.New("arr")
					sliceInit := emptySlice(result, elemType.GoType())

					elem := it.New("elem")
					collectBody := []ast.Stmt{
						bst.Assign(result, bst.Call(nil, "append", result, elem)),
					}

					callback := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: rcvr.Expr, Sel: ast.NewIdent("Each")},
							Args: []ast.Expr{
								&ast.FuncLit{
									Type: &ast.FuncType{
										Params: &ast.FieldList{
											List: []*ast.Field{{
												Names: []*ast.Ident{elem},
												Type:  ast.NewIdent(elemType.GoType()),
											}},
										},
									},
									Body: &ast.BlockStmt{List: collectBody},
								},
							},
						},
					}

					return Transform{
						Expr:  result,
						Stmts: []ast.Stmt{sliceInit, callback},
					}
				},
			})
		},
	})
}

// replaceIdent replaces all occurrences of oldIdent with newIdent in statements.
func replaceIdent(stmts []ast.Stmt, oldIdent, newIdent *ast.Ident) {
	for _, stmt := range stmts {
		ast.Inspect(stmt, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				if ident.Name == oldIdent.Name {
					ident.Name = newIdent.Name
				}
			}
			return true
		})
	}
}

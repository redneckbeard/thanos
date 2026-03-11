package types

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Range struct {
	Element  Type
	Instance instance
}

var RangeClass = NewClass("Range", "Object", nil, ClassRegistry)

func NewRange(inner Type) Type {
	return Range{Element: inner, Instance: RangeClass.Instance}
}

func (t Range) Equals(t2 Type) bool { return t == t2 }
func (t Range) String() string      { return fmt.Sprintf("Range(%s)", t.Element) }
func (t Range) GoType() string      { return fmt.Sprintf("*stdlib.Range[%s]", t.Element.GoType()) }
func (t Range) IsComposite() bool   { return true }
func (t Range) Outer() Type         { return Range{} }
func (t Range) Inner() Type         { return t.Element }
func (t Range) ClassName() string   { return "Range" }
func (t Range) IsMultiple() bool    { return false }

func (t Range) HasMethod(m string) bool {
	return t.Instance.HasMethod(m)
}

func (t Range) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.Instance.MustResolve(m).ReturnType(t, b, args)
}

func (t Range) BlockArgTypes(m string, args []Type) []Type {
	return t.Instance.MustResolve(m).blockArgs(t, args)
}

func (t Range) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.Instance.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Range) Resolve(m string) (MethodSpec, bool) {
	return t.Instance.Resolve(m)
}

func (t Range) GetMethodSpec(m string) (MethodSpec, bool) {
	return t.Instance.Resolve(m)
}

func (t Range) MustResolve(m string) MethodSpec {
	return t.Instance.MustResolve(m)
}

// rangeComponents extracts lower, upper, and inclusive from a CompositeLit range expression.
// Returns (lower, upper, inclusive, ok).
func rangeComponents(rcvr ast.Expr) (lower, upper ast.Expr, inclusive bool, ok bool) {
	rangeExpr, isLit := rcvr.(*ast.CompositeLit)
	if !isLit {
		return nil, nil, false, false
	}
	lower = rangeExpr.Elts[0]
	upper = rangeExpr.Elts[1]
	inclusive = rangeExpr.Elts[2].(*ast.Ident).Name == "true"
	return lower, upper, inclusive, true
}

// rangeForLoop builds a for-loop: for i := lower; i </<= upper; i++ { body }
func rangeForLoop(i *ast.Ident, lower, upper ast.Expr, inclusive bool, body []ast.Stmt) *ast.ForStmt {
	cmpTok := token.LSS
	if inclusive {
		cmpTok = token.LEQ
	}
	return &ast.ForStmt{
		Init: bst.Define(i, lower),
		Cond: bst.Binary(i, cmpTok, upper),
		Post: &ast.IncDecStmt{X: i, Tok: token.INC},
		Body: &ast.BlockStmt{List: body},
	}
}

func init() {
	RangeClass.Instance.Def("===", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			upperTok := token.LSS
			var lower, upper ast.Expr
			if rangeExpr, ok := rcvr.Expr.(*ast.CompositeLit); ok {
				lower, upper = rangeExpr.Elts[0], rangeExpr.Elts[1]
				if rangeExpr.Elts[2].(*ast.Ident).Name == "true" {
					upperTok = token.LEQ
				}
				return Transform{
					Expr: bst.Binary(
						bst.Binary(args[0].Expr, token.GEQ, lower),
						token.LAND,
						bst.Binary(args[0].Expr, upperTok, upper),
					),
				}
			}
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Covers", args[0].Expr),
			}
		},
	})

	// each: (1..5).each { |i| ... }
	RangeClass.Instance.Def("each", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Range).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			i := blk.Args[0]
			loop := rangeForLoop(i.(*ast.Ident), lower, upper, inclusive, blk.Statements)
			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{loop},
			}
		},
	})

	// to_a: (1..5).to_a → []int{1, 2, 3, 4, 5}
	RangeClass.Instance.Def("to_a", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r.(Range).Element), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{
					Expr:    bst.Call(rcvr.Expr, "ToA"),
					Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
				}
			}
			result := it.New("rangeArr")
			elemType := rcvr.Type.(Range).Element.GoType()
			init := emptySlice(result, elemType)
			i := it.New("i")
			loop := rangeForLoop(i, lower, upper, inclusive, []ast.Stmt{
				bst.Assign(result, bst.Call(nil, "append", result, i)),
			})
			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{init, loop},
			}
		},
	})

	// map: (1..5).map { |i| i * 2 }
	RangeClass.Instance.Def("map", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Range).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(b), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			mapped := it.New("mapped")
			init := emptySlice(mapped, blk.ReturnType.GoType())
			rewriteReturnsToAppend(blk.Statements, mapped)
			i := blk.Args[0]
			loop := rangeForLoop(i.(*ast.Ident), lower, upper, inclusive, blk.Statements)
			return Transform{
				Expr:  mapped,
				Stmts: []ast.Stmt{init, loop},
			}
		},
	})

	// select: (1..5).select { |i| i > 3 }
	RangeClass.Instance.Def("select", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Range).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r.(Range).Element), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			selected := it.New("selected")
			elemType := rcvr.Type.(Range).Element.GoType()
			init := emptySlice(selected, elemType)
			i := blk.Args[0]

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			appendStmt := bst.Assign(
				selected,
				bst.Call(nil, "append", selected, i),
			)
			ifStmt := &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{List: []ast.Stmt{appendStmt}},
			}
			blk.Statements[len(blk.Statements)-1] = ifStmt

			loop := rangeForLoop(i.(*ast.Ident), lower, upper, inclusive, blk.Statements)
			return Transform{
				Expr:  selected,
				Stmts: []ast.Stmt{init, loop},
			}
		},
	})

	// include?: (1..10).include?(5)
	RangeClass.Instance.Def("include?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{
					Expr: bst.Call(rcvr.Expr, "Covers", args[0].Expr),
				}
			}
			upperTok := token.LSS
			if inclusive {
				upperTok = token.LEQ
			}
			return Transform{
				Expr: bst.Binary(
					bst.Binary(args[0].Expr, token.GEQ, lower),
					token.LAND,
					bst.Binary(args[0].Expr, upperTok, upper),
				),
			}
		},
	})

	// size/length: (1..10).size
	sizeSpec := MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: bst.Call(rcvr.Expr, "Size")}
			}
			// size = upper - lower (+ 1 if inclusive)
			diff := bst.Binary(upper, token.SUB, lower)
			if inclusive {
				diff = bst.Binary(diff, token.ADD, bst.Int(1))
			}
			return Transform{Expr: diff}
		},
	}
	RangeClass.Instance.Def("size", sizeSpec)
	RangeClass.Instance.Def("length", sizeSpec)

	// min: (3..7).min → 3
	RangeClass.Instance.Def("min", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Range).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, _, _, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: bst.Call(rcvr.Expr, "Lower")}
			}
			return Transform{Expr: lower}
		},
	})

	// max: (3..7).max → 7
	RangeClass.Instance.Def("max", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Range).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			_, upper, _, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: bst.Call(rcvr.Expr, "Upper")}
			}
			return Transform{Expr: upper}
		},
	})

	// first: (1..10).first → 1
	RangeClass.Instance.Def("first", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Range).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, _, _, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: bst.Call(rcvr.Expr, "Lower")}
			}
			return Transform{Expr: lower}
		},
	})

	// last: (1..10).last → 10
	RangeClass.Instance.Def("last", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Range).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			_, upper, _, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: bst.Call(rcvr.Expr, "Upper")}
			}
			return Transform{Expr: upper}
		},
	})

	// sum: (1..100).sum → arithmetic formula: n*(a+b)/2
	RangeClass.Instance.Def("sum", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Range).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: bst.Call(rcvr.Expr, "Sum")}
			}
			// sum via loop for simplicity
			sum := it.New("sum")
			i := it.New("i")
			accum := &ast.AssignStmt{
				Lhs: []ast.Expr{sum},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{i},
			}
			loop := rangeForLoop(i, lower, upper, inclusive, []ast.Stmt{accum})
			init := bst.Define(sum, bst.Int(0))
			return Transform{
				Expr:  sum,
				Stmts: []ast.Stmt{init, loop},
			}
		},
	})

	// any?: (1..10).any? { |i| i > 5 }
	RangeClass.Instance.Def("any?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Range).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			result := it.New("any")
			init := bst.Define(result, ast.NewIdent("false"))
			i := blk.Args[0]

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			setTrue := bst.Assign(result, ast.NewIdent("true"))
			breakStmt := &ast.BranchStmt{Tok: token.BREAK}
			ifStmt := &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{List: []ast.Stmt{setTrue, breakStmt}},
			}
			blk.Statements[len(blk.Statements)-1] = ifStmt

			loop := rangeForLoop(i.(*ast.Ident), lower, upper, inclusive, blk.Statements)
			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{init, loop},
			}
		},
	})

	// none?: (1..10).none? { |i| i > 20 }
	RangeClass.Instance.Def("none?", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Range).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			result := it.New("none")
			init := bst.Define(result, ast.NewIdent("true"))
			i := blk.Args[0]

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			setFalse := bst.Assign(result, ast.NewIdent("false"))
			breakStmt := &ast.BranchStmt{Tok: token.BREAK}
			ifStmt := &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{List: []ast.Stmt{setFalse, breakStmt}},
			}
			blk.Statements[len(blk.Statements)-1] = ifStmt

			loop := rangeForLoop(i.(*ast.Ident), lower, upper, inclusive, blk.Statements)
			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{init, loop},
			}
		},
	})

	// reduce/inject: (1..5).reduce(0) { |sum, i| sum + i }
	reduceSpec := MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{args[0], r.(Range).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return args[0], nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			acc := blk.Args[0]
			accInit := bst.Define(acc, args[0].Expr)

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = bst.Assign(acc, finalStatement.Results[0])

			i := blk.Args[1]
			loop := rangeForLoop(i.(*ast.Ident), lower, upper, inclusive, blk.Statements)
			return Transform{
				Expr:  acc,
				Stmts: []ast.Stmt{accInit, loop},
			}
		},
	}
	RangeClass.Instance.Def("reduce", reduceSpec)
	RangeClass.Instance.Def("inject", reduceSpec)

	// each_with_index: (10..13).each_with_index { |val, idx| ... }
	RangeClass.Instance.Def("each_with_index", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Range).Element, IntType}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stripBlockReturn(blk)
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			val := blk.Args[0]
			idx := blk.Args[1]
			// idx := 0 before loop, idx++ at end of loop body
			idxInit := bst.Define(idx, bst.Int(0))
			idxInc := &ast.IncDecStmt{X: idx.(ast.Expr), Tok: token.INC}
			body := append(blk.Statements, idxInc)
			loop := rangeForLoop(val.(*ast.Ident), lower, upper, inclusive, body)
			return Transform{
				Expr:  rcvr.Expr,
				Stmts: []ast.Stmt{idxInit, loop},
			}
		},
	})

	// reject: (1..10).reject { |i| i % 2 == 0 }
	RangeClass.Instance.Def("reject", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Range).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NewArray(r.(Range).Element), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			rejected := it.New("rejected")
			elemType := rcvr.Type.(Range).Element.GoType()
			init := emptySlice(rejected, elemType)
			i := blk.Args[0]

			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			appendStmt := bst.Assign(rejected, bst.Call(nil, "append", rejected, i))
			ifStmt := &ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: finalStatement.Results[0]},
				Body: &ast.BlockStmt{List: []ast.Stmt{appendStmt}},
			}
			blk.Statements[len(blk.Statements)-1] = ifStmt

			loop := rangeForLoop(i.(*ast.Ident), lower, upper, inclusive, blk.Statements)
			return Transform{
				Expr:  rejected,
				Stmts: []ast.Stmt{init, loop},
			}
		},
	})

	// find/detect: (1..100).find { |i| i > 50 }
	findSpec := MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r.(Range).Element}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r.(Range).Element, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			lower, upper, inclusive, ok := rangeComponents(rcvr.Expr)
			if !ok {
				return Transform{Expr: rcvr.Expr}
			}
			result := it.New("found")
			elemType := rcvr.Type.(Range).Element.GoType()
			decl := bst.Define(result, bst.Int(0))
			_ = elemType

			i := blk.Args[0]
			finalStatement := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			body := &ast.IfStmt{
				Cond: finalStatement.Results[0],
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						bst.Assign(result, i),
						&ast.BranchStmt{Tok: token.BREAK},
					},
				},
			}
			blk.Statements[len(blk.Statements)-1] = body

			loop := rangeForLoop(i.(*ast.Ident), lower, upper, inclusive, blk.Statements)
			return Transform{
				Expr:  result,
				Stmts: []ast.Stmt{decl, loop},
			}
		},
	}
	RangeClass.Instance.Def("find", findSpec)
	RangeClass.Instance.Def("detect", findSpec)
}

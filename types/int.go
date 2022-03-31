package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type Int struct {
	*proto
}

var IntType = Int{newProto("Integer", "Numeric", ClassRegistry)}

var IntClass = NewClass("Integer", "Numeric", IntType, ClassRegistry)

func (t Int) Equals(t2 Type) bool { return t == t2 }
func (t Int) String() string      { return "IntType" }
func (t Int) GoType() string      { return "int" }
func (t Int) IsComposite() bool   { return false }

func (t Int) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.MustResolve(m).ReturnType(t, b, args)
}

//TODO we don't need this in the interface. Instead, the parser or compiler should retrieve the MethodSpec and check for a not-nil blockArgs (which will then need to be exported
func (t Int) BlockArgTypes(m string, args []Type) []Type {
	return t.MustResolve(m).blockArgs(t, args)
}

func (t Int) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.MustResolve(m).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Int) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Int) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Int) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t Int) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func integerOperatorSpec(tok token.Token) MethodSpec {
	return MethodSpec{
		ReturnType: func(rcvr Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, tok, args[0].Expr),
			}
		},
	}
}

func init() {
	IntType.Def("&", integerOperatorSpec(token.AND))
	IntType.Def("|", integerOperatorSpec(token.OR))
	IntType.Def("^", integerOperatorSpec(token.XOR))
	IntType.Def("<<", integerOperatorSpec(token.SHL))
	IntType.Def(">>", integerOperatorSpec(token.SHR))
	IntType.Def("[]", MethodSpec{
		ReturnType: func(rcvr Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// Ruby lets you access bit value by index. The indexes, somewhat
			// confusingly, are 0-based positions by least-significant bit. We can
			// recreate this with bit shift operators like so:
			//   (i & (1 << index)) >> index

			return Transform{
				Expr: bst.Binary(
					bst.Binary(
						rcvr.Expr,
						token.AND,
						bst.Binary(bst.Int(1), token.SHL, args[0].Expr),
					),
					token.SHR,
					args[0].Expr,
				),
			}
		},
	})
	//`Integer#+@`
	//`Integer#-@`
	//`Integer#[]`
	//`Integer#^`
	//TODO why does this break? IntType.Alias("magnitude", "abs")
	//`Integer#abs2`
	//`Integer#allbits?`
	//`Integer#angle`
	//`Integer#anybits?`
	//`Integer#arg`
	//`Integer#between?`
	//`Integer#bit_length`
	//`Integer#ceil`
	//`Integer#chr`
	IntType.Def("chr", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(nil, "string", bst.Call(nil, "rune", rcvr.Expr)),
			}
		},
	})
	//`Integer#clamp`
	//`Integer#coerce`
	//`Integer#conj`
	//`Integer#conjugate`
	//`Integer#denominator`
	//`Integer#digits`
	//`Integer#div`
	//`Integer#divmod`
	IntType.Def("downto", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			blockVar := blk.Args[0]
			upper, lower := rcvr.Expr, args[0].Expr
			loop := &ast.ForStmt{
				Init: bst.Define(blockVar, upper),
				Cond: bst.Binary(blockVar, token.GEQ, lower),
				Post: &ast.IncDecStmt{X: blockVar, Tok: token.DEC},
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
	IntType.Def("even?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(bst.Binary(rcvr.Expr, token.REM, bst.Int(2)), token.EQL, bst.Int(0)),
			}
		},
	})
	//`Integer#fdiv`
	//`Integer#finite?`
	//`Integer#floor`
	//`Integer#gcd`
	//`Integer#gcdlcm`
	//`Integer#i`
	//`Integer#imag`
	//`Integer#imaginary`
	//`Integer#infinite?`
	IntType.Def("integer?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: it.Get("true"),
			}
		},
	})
	//`Integer#lcm`
	//`Integer#modulo`
	//`Integer#next`
	//`Integer#nobits?`
	//`Integer#nonzero?`
	//`Integer#numerator`
	IntType.Def("odd?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(bst.Binary(rcvr.Expr, token.REM, bst.Int(2)), token.EQL, bst.Int(1)),
			}
		},
	})
	//`Integer#ord`
	//`Integer#phase`
	//`Integer#polar`
	//`Integer#pow`
	//`Integer#pred`
	//`Integer#quo`
	//`Integer#rationalize`
	//`Integer#real`
	//`Integer#real?`
	//`Integer#rect`
	//`Integer#rectangular`
	//`Integer#remainder`
	//`Integer#round`
	//`Integer#singleton_method_added`
	//`Integer#size`
	//`Integer#step`
	//`Integer#succ`
	IntType.Def("times", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			blockVar := blk.Args[0]
			loop := &ast.ForStmt{
				Init: bst.Define(blockVar, bst.Int(0)),
				Cond: bst.Binary(blockVar, token.LSS, rcvr.Expr),
				Post: &ast.IncDecStmt{X: blockVar, Tok: token.INC},
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
	//`Integer#to_c`
	//`Integer#to_f`
	//`Integer#to_i`
	//`Integer#to_int`
	//`Integer#to_r`
	IntType.Def("to_s", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) == 0 {
				return Transform{
					Expr:    bst.Call("strconv", "Itoa", rcvr.Expr),
					Imports: []string{"strconv"},
				}
			}
			return Transform{
				Expr:    bst.Call("strconv", "FormatInt", rcvr.Expr, args[0].Expr),
				Imports: []string{"strconv"},
			}
		},
	})
	//`Integer#truncate`
	//`Integer#upto`
	IntType.Def("upto", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			blockVar := blk.Args[0]
			lower, upper := rcvr.Expr, args[0].Expr
			loop := &ast.ForStmt{
				Init: bst.Define(blockVar, lower),
				Cond: bst.Binary(blockVar, token.LEQ, upper),
				Post: &ast.IncDecStmt{X: blockVar, Tok: token.INC},
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
}

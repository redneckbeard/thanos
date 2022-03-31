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
	// `String#+@`
	// `String#-@`
	// `String#[]`
	// `String#[]=`
	// `String#ascii_only?`
	// `String#b`
	// `String#between?`
	// `String#bytes`
	// `String#bytesize`
	// `String#byteslice`
	StringType.Def("capitalize", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("strings", "Title", rcvr.Expr, args[0].Expr),
				Imports: []string{"strings"},
			}
		},
	})
	// `String#capitalize!`
	// `String#casecmp`
	// `String#casecmp?`
	// `String#center`
	// `String#chars`
	// `String#chomp`
	// `String#chomp!`
	// `String#chop`
	// `String#chop!`
	// `String#chr`
	// `String#clamp`
	// `String#clear`
	// `String#codepoints`
	// `String#concat`
	// `String#count`
	// `String#crypt`
	// `String#delete`
	// `String#delete!`
	// `String#delete_prefix`
	// `String#delete_prefix!`
	// `String#delete_suffix`
	// `String#delete_suffix!`
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
	// `String#downcase!`
	// `String#dump`
	// `String#each_byte`
	// `String#each_char`
	// `String#each_codepoint`
	// `String#each_grapheme_cluster`
	// `String#each_line`
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
	// `String#encode`
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
	// `String#getbyte`
	// `String#grapheme_clusters`
	StringType.Def("gsub", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if blk != nil {
				panic("Block arguments not yet supported for gsub")
			}
			if _, ok := args[1].Type.(Hash); ok {
				panic("Hash arguments not yet supported for gsub")
			}
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
	// `String#gsub!`
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
	// `String#index`
	// `String#insert`
	// `String#intern`
	// `String#length`
	// `String#lines`
	// `String#ljust`
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
	// `String#lstrip!`
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
	// `String#match?`
	// `String#next`
	// `String#next!`
	// `String#oct`
	// `String#ord`
	// `String#partition`
	// `String#prepend`
	// `String#replace`
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
	// `String#rindex`
	// `String#rjust`
	// `String#rpartition`
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
	// `String#rstrip!`
	// `String#scan`
	// `String#scrub`
	// `String#scrub!`
	// `String#setbyte`
	StringType.Def("size", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
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
	// `String#squeeze`
	// `String#squeeze!`
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
	// `String#strip!`
	// `String#sub`
	// `String#sub!`
	// `String#succ`
	// `String#succ!`
	// `String#sum`
	// `String#swapcase`
	// `String#swapcase!`
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
	// `String#tr`
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
	// `String#upcase!`
	// `String#upto`
	// `String#valid_encoding?`

}

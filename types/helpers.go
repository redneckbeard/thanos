package types

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

var NoopReturnSelf = MethodSpec{
	ReturnType: func(r Type, b Type, args []Type) (Type, error) {
		return r, nil
	},
	TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
		return Transform{
			Expr: rcvr.Expr,
		}
	},
}

var AlwaysTrue = MethodSpec{
	ReturnType: func(r Type, b Type, args []Type) (Type, error) {
		return BoolType, nil
	},
	TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
		return Transform{
			Expr: it.Get("true"),
		}
	},
}

var AlwaysFalse = MethodSpec{
	ReturnType: func(r Type, b Type, args []Type) (Type, error) {
		return BoolType, nil
	},
	TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
		return Transform{
			Expr: it.Get("false"),
		}
	},
}

func emptySlice(name *ast.Ident, inner string) *ast.AssignStmt {
	targetSlice := &ast.CompositeLit{
		Type: &ast.ArrayType{
			Elt: ast.NewIdent(inner),
		},
		Elts: []ast.Expr{},
	}
	targetSliceVarInit := bst.Define(name, targetSlice)
	return targetSliceVarInit
}

func appendLoop(loopVar, appendTo, rangeOver, appendHead, appendTail ast.Expr) *ast.RangeStmt {
	appendStmt := bst.Assign(appendTo, bst.Call(nil, "append", appendHead, appendTail))

	return &ast.RangeStmt{
		Key:   ast.NewIdent("_"),
		Value: loopVar,
		Tok:   token.DEFINE,
		X:     rangeOver,
		Body: &ast.BlockStmt{
			List: []ast.Stmt{appendStmt},
		},
	}
}

func UnwrapTypeExprs(typeExprs []TypeExpr) []ast.Expr {
	exprs := []ast.Expr{}
	for _, typeExpr := range typeExprs {
		exprs = append(exprs, typeExpr.Expr)
	}
	return exprs
}

func simpleComparisonOperatorSpec(tok token.Token) MethodSpec {
	return MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			if r == args[0] {
				return BoolType, nil
			}
			return nil, fmt.Errorf("Tried to compare disparate types %s and %s", r, args[0])
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, tok, args[0].Expr),
			}
		},
	}
}

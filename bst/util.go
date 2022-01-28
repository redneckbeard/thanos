package bst

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

func Call(rcvr interface{}, method interface{}, args ...ast.Expr) *ast.CallExpr {
	var fun ast.Expr
	if rcvr == nil {
		fun = toExpr(method)
	} else {
		fun = Dot(rcvr, method)
	}
	return &ast.CallExpr{
		Fun:  fun,
		Args: args,
	}
}

func Binary(left ast.Expr, op token.Token, right ast.Expr) ast.Expr {
	return &ast.BinaryExpr{
		X:  left,
		Op: op,
		Y:  right,
	}
}

func Dot(obj, member interface{}) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   toExpr(obj),
		Sel: toExpr(member).(*ast.Ident),
	}
}

func toExpr(i interface{}) ast.Expr {
	switch x := i.(type) {
	case string:
		return ast.NewIdent(x)
	case ast.Expr:
		return x
	default:
		panic("only supported types are string and ast.Expr")
	}
}

func String(s string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.STRING,
		Value: fmt.Sprintf(`"%s"`, s),
	}
}

func Int(i interface{}) *ast.BasicLit {
	var val string
	if str, ok := i.(string); ok {
		val = str
	} else {
		val = strconv.Itoa(i.(int))
	}
	return &ast.BasicLit{
		Kind:  token.INT,
		Value: val,
	}
}

type AssignFunc func(interface{}, interface{}) *ast.AssignStmt

var opAssignTokens = map[string]token.Token{
	"+":  token.ADD_ASSIGN,
	"-":  token.SUB_ASSIGN,
	"*":  token.MUL_ASSIGN,
	"/":  token.QUO_ASSIGN,
	"%":  token.REM_ASSIGN,
	"&":  token.AND_ASSIGN,
	"|":  token.OR_ASSIGN,
	"^":  token.XOR_ASSIGN,
	"<":  token.SHL_ASSIGN,
	">":  token.SHR_ASSIGN,
	"&^": token.AND_NOT_ASSIGN,
}

func OpAssign(op string) AssignFunc {
	return func(lhs, rhs interface{}) *ast.AssignStmt {
		return &ast.AssignStmt{
			Lhs: toExprSlice(lhs),
			Tok: opAssignTokens[op],
			Rhs: toExprSlice(rhs),
		}
	}
}

func Assign(lhs, rhs interface{}) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: toExprSlice(lhs),
		Tok: token.ASSIGN,
		Rhs: toExprSlice(rhs),
	}
}

func Define(lhs, rhs interface{}) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: toExprSlice(lhs),
		Tok: token.DEFINE,
		Rhs: toExprSlice(rhs),
	}
}

func Declare(kind token.Token, name *ast.Ident, goType ast.Expr) ast.Decl {
	return &ast.GenDecl{
		Tok: kind,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{name},
				Type:  goType,
			},
		},
	}
}

func toExprSlice(i interface{}) []ast.Expr {
	if slice, ok := i.([]ast.Expr); ok {
		return slice
	} else {
		return []ast.Expr{i.(ast.Expr)}
	}
}

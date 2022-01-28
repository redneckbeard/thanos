package types

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

type ReturnTypeSpec func(left, right Type) (Type, error)
type ASTTransformSpec func(left, right TypeExpr, tok token.Token) Transform

type OperatorSpec struct {
	ValidOperandTypes []Type
	ComputeReturnType ReturnTypeSpec
	TransformAST      ASTTransformSpec
}

func (o OperatorSpec) IsValidOperandType(t Type) bool {
	if t == nil {
		return false
	}
	var outer Type
	if t.IsComposite() {
		outer = t.(CompositeType).Outer()
	} else {
		outer = t
	}
	for _, v := range o.ValidOperandTypes {
		if v == AnyType {
			return true
		}
		if v == outer {
			return true
		}
	}
	return false
}

var nativeOperator OperatorSpec = OperatorSpec{
	ValidOperandTypes: []Type{IntType, FloatType},
	ComputeReturnType: func(left, right Type) (Type, error) {
		if left == right {
			return left, nil
		}
		if left == FloatType || right == FloatType {
			return FloatType, nil
		}
		return IntType, nil
	},
	TransformAST: func(left, right TypeExpr, tok token.Token) Transform {
		leftExpr, rightExpr := left.Expr, right.Expr
		if left.Type == FloatType && right.Type == IntType {
			if _, ok := right.Expr.(*ast.BasicLit); !ok {
				rightExpr = bst.Call(nil, "float64", rightExpr)
			}
		} else if left.Type == IntType && right.Type == FloatType {
			if _, ok := left.Expr.(*ast.BasicLit); !ok {
				leftExpr = bst.Call(nil, "float64", leftExpr)
			}
		}
		return Transform{
			Expr: bst.Binary(leftExpr, tok, rightExpr),
		}
	},
}

var addOperator OperatorSpec = OperatorSpec{
	ValidOperandTypes: []Type{IntType, FloatType, StringType},
	ComputeReturnType: func(left, right Type) (Type, error) {
		if left == right {
			return left, nil
		}
		if left == FloatType || right == FloatType {
			return FloatType, nil
		}
		return IntType, nil
	},
}

var powOperator OperatorSpec = OperatorSpec{
	ValidOperandTypes: []Type{IntType, FloatType},
	ComputeReturnType: func(left, right Type) (Type, error) {
		if left == IntType && right == IntType {
			return IntType, nil
		}
		return FloatType, nil
	},
	TransformAST: func(left, right TypeExpr, tok token.Token) Transform {
		leftExpr, rightExpr := left.Expr, right.Expr
		if _, ok := left.Expr.(*ast.BasicLit); !ok && left.Type == IntType {
			leftExpr = bst.Call(nil, "float64", leftExpr)
		}
		if _, ok := right.Expr.(*ast.BasicLit); !ok && right.Type == IntType {
			rightExpr = bst.Call(nil, "float64", rightExpr)
		}
		expr := bst.Call("math", "Pow", leftExpr, rightExpr)
		if left.Type == IntType && right.Type == IntType {
			expr = bst.Call(nil, "int", expr)
		}
		return Transform{
			Expr: expr,
		}
	},
}

var comparisonOperator OperatorSpec = OperatorSpec{
	ValidOperandTypes: []Type{IntType, FloatType, StringType},
	ComputeReturnType: func(left, right Type) (Type, error) {
		if left == right {
			return BoolType, nil
		}
		return nil, fmt.Errorf("Tried to compare disparate types %s and %s", left, right)
	},
}

var logicalOperator OperatorSpec = OperatorSpec{
	ValidOperandTypes: []Type{IntType, FloatType, StringType, BoolType},
	ComputeReturnType: func(left, right Type) (Type, error) {
		if left == right {
			return BoolType, nil
		}
		return nil, fmt.Errorf("Tried to compare disparate types %s and %s", left, right)
	},
}

var matchOperator = OperatorSpec{
	ValidOperandTypes: []Type{StringType, RegexpType},
	ComputeReturnType: func(left, right Type) (Type, error) {
		// In reality the match operator returns an int, or nil if there's no match. However, in practical
		// use it is relied on for evaluation to a boolean
		return BoolType, nil
	},
	TransformAST: func(left, right TypeExpr, tok token.Token) Transform {
		// figure out which one is the regexp
		// need imports like on method Transforms
		// Transforms need way to add global declarations, i.e. if no interps, here, we should use MustCompile
		var regexp, str ast.Expr
		if left.Type == RegexpType {
			regexp, str = left.Expr, right.Expr
		} else {
			regexp, str = right.Expr, left.Expr
		}
		return Transform{
			Expr: bst.Call(regexp, "MatchString", str),
		}
	},
}

var Operators = map[string]struct {
	Spec    OperatorSpec
	GoToken token.Token
}{
	"+":  {addOperator, token.ADD},
	"-":  {nativeOperator, token.SUB},
	"*":  {nativeOperator, token.MUL},
	"/":  {nativeOperator, token.QUO},
	"%":  {nativeOperator, token.REM},
	"<<": {nativeOperator, token.SHL},
	"**": {powOperator, token.ILLEGAL},
	"<":  {comparisonOperator, token.LSS},
	">":  {comparisonOperator, token.GTR},
	"<=": {comparisonOperator, token.LEQ},
	">=": {comparisonOperator, token.GEQ},
	"==": {comparisonOperator, token.EQL},
	"!=": {comparisonOperator, token.NEQ},
	"&&": {logicalOperator, token.LAND},
	"||": {logicalOperator, token.LOR},
	"=~": {matchOperator, token.ILLEGAL},
}

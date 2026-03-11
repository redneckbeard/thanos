package compiler

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

// receiverIsOrderSafe checks if a MethodCall's receiver is a hash variable
// that has been marked as order-safe (can use native map).
func (g *GoProgram) receiverIsOrderSafe(receiver parser.Node) bool {
	if ident, ok := receiver.(*parser.IdentNode); ok {
		return g.isOrderSafe(ident.Val)
	}
	return false
}

// nativeMapTransform generates Go AST for hash method calls using native
// map[K]V operations instead of *stdlib.OrderedMap methods. Returns the
// transform and true if handled, or zero Transform and false if the method
// should fall through to the regular OrderedMap transform.
func (g *GoProgram) nativeMapTransform(rcvr ast.Expr, rcvrType types.Type, methodName string, args []types.TypeExpr, blk *types.Block) (types.Transform, bool) {
	h, ok := rcvrType.(types.Hash)
	if !ok {
		return types.Transform{}, false
	}

	switch methodName {
	case "[]":
		// h.Data[key] → h[key]
		return types.Transform{
			Expr: &ast.IndexExpr{X: rcvr, Index: args[0].Expr},
		}, true

	case "[]=":
		// h.Set(key, val) → h[key] = val
		return types.Transform{
			Stmts: []ast.Stmt{
				bst.Assign(
					&ast.IndexExpr{X: rcvr, Index: args[0].Expr},
					args[1].Expr,
				),
			},
		}, true

	case "length", "size":
		// h.Len() → len(h)
		return types.Transform{
			Expr: bst.Call(nil, "len", rcvr),
		}, true

	case "empty?":
		// h.Len() == 0 → len(h) == 0
		return types.Transform{
			Expr: bst.Binary(bst.Call(nil, "len", rcvr), token.EQL, bst.Int(0)),
		}, true

	case "clear":
		// h.Clear() → clear(h) (Go 1.21+)
		return types.Transform{
			Stmts: []ast.Stmt{
				&ast.ExprStmt{X: bst.Call(nil, "clear", rcvr)},
			},
		}, true

	case "has_key?", "key?", "include?", "member?":
		// h.HasKey(key) → _, ok := h[key]; ok
		ok := g.it.New("ok")
		return types.Transform{
			Stmts: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{g.it.Get("_"), ok},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{&ast.IndexExpr{X: rcvr, Index: args[0].Expr}},
				},
			},
			Expr: ok,
		}, true

	default:
		_ = h
		return types.Transform{}, false
	}
}

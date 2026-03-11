package compiler

import (
	"go/ast"

	"github.com/redneckbeard/thanos/parser"
	"github.com/redneckbeard/thanos/types"
)

func (g *GoProgram) TransformMethodCall(c *parser.MethodCall) types.Transform {
	var blk *types.Block
	if c.Block != nil {
		blk = g.BuildBlock(c.Block)
	}
	return g.getTransform(c, g.CompileExpr(c.Receiver), c.Receiver.Type(), c.MethodName, c.Args, blk, false)
}

func (g *GoProgram) TransformMethodCallStmt(c *parser.MethodCall) types.Transform {
	var blk *types.Block
	if c.Block != nil {
		blk = g.BuildBlock(c.Block)
	}
	return g.getTransform(c, g.CompileExpr(c.Receiver), c.Receiver.Type(), c.MethodName, c.Args, blk, true)
}

func (g *GoProgram) getTransform(call *parser.MethodCall, rcvr ast.Expr, rcvrType types.Type, methodName string, args parser.ArgsNode, blk *types.Block, stmtContext bool) types.Transform {
	var argExprs []types.TypeExpr
	if call != nil && call.Method != nil {
		argExprs = g.CompileArgs(call, args)
	} else {
		// Check if the MethodSpec has KwargsSpec — if so, partition args
		// into positional and kwargs, appending kwargs in declared order.
		spec, hasSpec := rcvrType.GetMethodSpec(methodName)
		if hasSpec && len(spec.KwargsSpec) > 0 {
			kwargMap := map[string]parser.Node{}
			var positional []parser.Node
			for _, a := range args {
				if kv, ok := a.(*parser.KeyValuePair); ok {
					kwargMap[kv.Label] = kv.Value
				} else {
					positional = append(positional, a)
				}
			}
			for _, a := range positional {
				argExprs = append(argExprs, types.TypeExpr{Expr: g.CompileExpr(a), Type: a.Type()})
			}
			for _, ks := range spec.KwargsSpec {
				if val, ok := kwargMap[ks.Name]; ok {
					argExprs = append(argExprs, types.TypeExpr{Expr: g.CompileExpr(val), Type: val.Type()})
				} else {
					argExprs = append(argExprs, types.TypeExpr{})
				}
			}
		} else {
			for _, a := range args {
				argExprs = append(argExprs, types.TypeExpr{Expr: g.CompileExpr(a), Type: a.Type()})
			}
		}
	}
	// For order-safe hashes, use native map operations
	if call != nil && call.Receiver != nil && g.receiverIsOrderSafe(call.Receiver) {
		if transform, ok := g.nativeMapTransform(rcvr, rcvrType, methodName, argExprs, blk); ok {
			g.AddImports(transform.Imports...)
			return transform
		}
	}
	if stmtContext {
		if spec, ok := rcvrType.GetMethodSpec(methodName); ok && spec.TransformStmtAST != nil {
			transform := spec.TransformStmtAST(types.TypeExpr{rcvrType, rcvr}, argExprs, blk, g.it)
			g.AddImports(transform.Imports...)
			g.localizeExpr(transform.Expr)
			return transform
		}
	}
	transform := rcvrType.TransformAST(
		methodName,
		rcvr,
		argExprs,
		blk,
		g.it,
	)
	g.AddImports(transform.Imports...)
	g.localizeExpr(transform.Expr)
	return transform
}

func (g *GoProgram) BuildBlock(blk *parser.Block) *types.Block {
	g.pushTracker()
	args := []ast.Expr{}
	argTypes := []types.Type{}
	for _, p := range blk.Params {
		if p.Kind == parser.Destructured {
			for _, nested := range p.Nested {
				args = append(args, g.it.Get(nested.Name))
				argTypes = append(argTypes, nested.Type())
			}
		} else {
			args = append(args, g.it.Get(p.Name))
			argTypes = append(argTypes, p.Type())
		}
	}
	g.newBlockStmt()
	g.State.Push(InBlockBody)
	defer func() {
		g.BlockStack.Pop()
		g.State.Pop()
	}()
	for _, s := range blk.Body.Statements {
		g.CompileStmt(s)
	}
	g.popTracker()
	return &types.Block{
		ReturnType: blk.Body.ReturnType,
		Args:       args,
		ArgTypes:   argTypes,
		Statements: g.BlockStack.Peek().List,
	}
}

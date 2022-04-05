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
	return g.getTransform(c, g.CompileExpr(c.Receiver), c.Receiver.Type(), c.MethodName, c.Args, blk)
}

func (g *GoProgram) getTransform(call *parser.MethodCall, rcvr ast.Expr, rcvrType types.Type, methodName string, args parser.ArgsNode, blk *types.Block) types.Transform {
	var argExprs []types.TypeExpr
	if call != nil && call.Method != nil {
		argExprs = g.CompileArgs(call, args)
	} else {
		for _, a := range args {
			argExprs = append(argExprs, types.TypeExpr{Expr: g.CompileExpr(a), Type: a.Type()})
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
	return transform
}

func (g *GoProgram) BuildBlock(blk *parser.Block) *types.Block {
	g.pushTracker()
	args := []ast.Expr{}
	for _, p := range blk.Params {
		args = append(args, g.it.Get(p.Name))
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
		Statements: g.BlockStack.Peek().List,
	}
}

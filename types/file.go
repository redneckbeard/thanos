package types

import (
	"go/ast"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/stdlib"
)

var FileType = NewClass("File", "Object", nil, ClassRegistry)

func init() {
	FileType.Instance.Def("initialize", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return FileType.Instance.(Type), nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var call *ast.CallExpr
			imports := []string{"os"}
			switch len(args) {
			case 1:
				call = bst.Call("os", "Open", args[0].Expr)
			case 2:
				call = bst.Call("os", "OpenFile", args[0].Expr)
				if lit, ok := args[1].Expr.(*ast.BasicLit); ok {
					mode := strings.Trim(lit.Value, `"`)
					flag, ok := stdlib.OpenModes[mode]
					if !ok {
						panic("Invalid mode: " + mode)
					}
					call.Args = append(call.Args, bst.Int(flag))
				} else {
					call.Args = append(call.Args, &ast.IndexExpr{
						X:     bst.Dot("stdlib", "OpenMode"),
						Index: args[1].Expr,
					})
					imports = append(imports, "github.com/redneckbeard/thanos/stdlib")
				}
				call.Args = append(call.Args, bst.Int("0666"))
			case 3:
				panic("File.new does not yet support an options hash")
			}
			file := it.New("f")
			stmt := bst.Define([]ast.Expr{file, it.Get("_")}, call)
			return Transform{
				Expr:    file,
				Stmts:   []ast.Stmt{stmt},
				Imports: imports,
			}
		},
	})

	FileType.Instance.Def("each", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return FileType, nil
		},
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{StringType}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// Ruby's File#each doesn't strip line terminators by default;
			// bufio.Scanner does, and we have to do something to soak up the
			// mismatch.  By relying here on a SplitFunc generator in stdlib, we
			// should be well positioned to support `chomp: true` when we get kwargs
			// support into the parser. Presently it will incorrectly work as
			// a positional arg.
			scanner := it.New("scanner")
			initScanner := bst.Define(scanner, bst.Call("bufio", "NewScanner", rcvr.Expr))
			makeSplitFuncArgs := UnwrapTypeExprs(args)
			if len(makeSplitFuncArgs) == 0 {
				makeSplitFuncArgs = []ast.Expr{bst.String(`\n`), it.Get("false")}
			}
			setSplit := &ast.ExprStmt{
				X: bst.Call(scanner, "Split", bst.Call("stdlib", "MakeSplitFunc", makeSplitFuncArgs...)),
			}
			lineDef := bst.Define(blk.Args, bst.Call(scanner, "Text"))
			loop := &ast.ForStmt{
				Cond: bst.Call(scanner, "Scan"),
				Body: &ast.BlockStmt{
					List: append([]ast.Stmt{lineDef}, blk.Statements...),
				},
			}
			return Transform{
				Expr:    rcvr.Expr,
				Stmts:   []ast.Stmt{initScanner, setSplit, loop},
				Imports: []string{"os", "bufio", "github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	FileType.Instance.Alias("each", "each_line")

	FileType.Instance.Def("close", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Close"),
			}
		},
	})

	FileType.Instance.Def("size", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			info := it.New("info")
			infoStmt := bst.Define([]ast.Expr{info, it.Get("_")}, bst.Call(rcvr.Expr, "Info"))
			return Transform{
				Stmts: []ast.Stmt{infoStmt},
				Expr:  bst.Call(info, "Size"),
			}
		},
	})

	FileType.Def("open", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			if blockReturnType != nil {
				return blockReturnType, nil
			}
			return FileType.Instance.(Type), nil
		},
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{FileType.Instance.(Type)}
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			newFile := rcvr.Type.TransformAST("new", rcvr.Expr, args, blk, it)
			if blk == nil {
				return newFile
			}
			// unlike a block that translates to a loop or other block-scoped construct, File.open solely
			// provides the convenience of not having to remember to close the file. Thus, we must set the
			// identifier provided with the block to be the one we returned when defining the file variable.
			f := newFile.Expr.(*ast.Ident)
			blockArg := blk.Args[0].(*ast.Ident)
			blockArg.Name = f.Name

			closeFile := FileType.Instance.MustResolve("close").TransformAST(TypeExpr{rcvr.Type, newFile.Expr}, []TypeExpr{}, nil, it)

			final := it.New("result")
			finalStmt := blk.Statements[len(blk.Statements)-1].(*ast.ReturnStmt)
			blk.Statements[len(blk.Statements)-1] = bst.Define(final, finalStmt.Results)

			stmts := append(newFile.Stmts, blk.Statements...)
			stmts = append(stmts, &ast.ExprStmt{X: closeFile.Expr})
			return Transform{
				Stmts:   stmts,
				Expr:    final,
				Imports: append(newFile.Imports, closeFile.Imports...),
			}
		},
	})

	FileType.Instance.Def("<<", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return receiverType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "WriteString", args[0].Expr),
			}
		},
	})
}

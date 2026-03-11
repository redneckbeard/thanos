package types

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/redneckbeard/thanos/bst"
	"github.com/redneckbeard/thanos/stdlib"
)

// panicOnErr generates: if errVar != nil { panic(errVar) }
func panicOnErr(errVar ast.Expr, it bst.IdentTracker) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: bst.Binary(errVar, token.NEQ, it.Get("nil")),
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{X: bst.Call(nil, "panic", errVar)},
			},
		},
	}
}

func fileOpenSetup(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) (Transform, *ast.CallExpr) {
	newFile := rcvr.Type.TransformAST("new", rcvr.Expr, args, blk, it)
	if blk != nil {
		f := newFile.Expr.(*ast.Ident)
		blockArg := blk.Args[0].(*ast.Ident)
		blockArg.Name = f.Name
	}
	closeFile := FileType.Instance.MustResolve("close").TransformAST(TypeExpr{rcvr.Type, newFile.Expr}, []TypeExpr{}, nil, it)
	return newFile, closeFile.Expr.(*ast.CallExpr)
}

func deferClose(fileExpr ast.Expr) *ast.DeferStmt {
	return &ast.DeferStmt{
		Call: bst.Call(fileExpr, "Close"),
	}
}

func fileOpenStmtTransform(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	newFile, _ := fileOpenSetup(rcvr, args, blk, it)
	if blk == nil {
		return newFile
	}
	stripBlockReturn(blk)
	stmts := append(newFile.Stmts, deferClose(newFile.Expr))
	stmts = append(stmts, blk.Statements...)
	return Transform{
		Stmts:   stmts,
		Imports: newFile.Imports,
	}
}

func fileOpenExprTransform(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	newFile, _ := fileOpenSetup(rcvr, args, blk, it)
	if blk == nil {
		return newFile
	}
	final := it.New("result")
	last := blk.Statements[len(blk.Statements)-1]
	if retStmt, ok := last.(*ast.ReturnStmt); ok {
		blk.Statements[len(blk.Statements)-1] = bst.Define(final, retStmt.Results)
	}
	stmts := append(newFile.Stmts, deferClose(newFile.Expr))
	stmts = append(stmts, blk.Statements...)
	return Transform{
		Stmts:   stmts,
		Expr:    final,
		Imports: newFile.Imports,
	}
}

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
			stripBlockReturn(blk)
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
		TransformStmtAST: fileOpenStmtTransform,
		TransformAST:     fileOpenExprTransform,
	})

	// File.read(path) → string(os.ReadFile(path)) with panic on err
	FileType.Def("read", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			data := it.New("data")
			err := it.New("err")
			readStmt := bst.Define([]ast.Expr{data, err}, bst.Call("os", "ReadFile", args[0].Expr))
			return Transform{
				Stmts:   []ast.Stmt{readStmt, panicOnErr(err, it)},
				Expr:    bst.Call(nil, "string", data),
				Imports: []string{"os"},
			}
		},
	})

	// File.write(path, content) → os.WriteFile(path, []byte(content), 0644)
	FileType.Def("write", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			bytes := &ast.CallExpr{
				Fun:  &ast.ArrayType{Elt: ast.NewIdent("byte")},
				Args: []ast.Expr{args[1].Expr},
			}
			err := it.New("err")
			writeStmt := bst.Define(err, bst.Call("os", "WriteFile", args[0].Expr, bytes, bst.Int("0644")))
			return Transform{
				Stmts:   []ast.Stmt{writeStmt, panicOnErr(err, it)},
				Expr:    bst.Call(nil, "len", bytes),
				Imports: []string{"os"},
			}
		},
	})

	// File.exist?(path) → stat, err check
	FileType.Def("exist?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "FileExists", args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})
	FileType.MakeAlias("exist?", "exists?", true)

	// File.directory?(path)
	FileType.Def("directory?", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("stdlib", "IsDirectory", args[0].Expr),
				Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
			}
		},
	})

	// File.basename(path) → filepath.Base(path)
	FileType.Def("basename", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("filepath", "Base", args[0].Expr),
				Imports: []string{"path/filepath"},
			}
		},
	})

	// File.dirname(path) → filepath.Dir(path)
	FileType.Def("dirname", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("filepath", "Dir", args[0].Expr),
				Imports: []string{"path/filepath"},
			}
		},
	})

	// File.extname(path) → filepath.Ext(path)
	FileType.Def("extname", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("filepath", "Ext", args[0].Expr),
				Imports: []string{"path/filepath"},
			}
		},
	})

	// File.delete(path) → os.Remove(path) with panic on err
	FileType.Def("delete", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			err := it.New("err")
			rmStmt := bst.Define(err, bst.Call("os", "Remove", args[0].Expr))
			return Transform{
				Stmts:   []ast.Stmt{rmStmt, panicOnErr(err, it)},
				Expr:    bst.Int(1),
				Imports: []string{"os"},
			}
		},
	})

	// File.size(path) → file stat size
	FileType.Def("size", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			info := it.New("info")
			err := it.New("err")
			statStmt := bst.Define([]ast.Expr{info, err}, bst.Call("os", "Stat", args[0].Expr))
			return Transform{
				Stmts:   []ast.Stmt{statStmt, panicOnErr(err, it)},
				Expr:    bst.Call(nil, "int", bst.Call(info, "Size")),
				Imports: []string{"os"},
			}
		},
	})

	// Instance method: File#write(str) → f.WriteString(str)
	FileType.Instance.Def("write", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return IntType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "WriteString", args[0].Expr),
			}
		},
	})

	// Instance method: File#read → io.ReadAll(f) as string
	FileType.Instance.Def("read", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			data := it.New("data")
			err := it.New("err")
			readStmt := bst.Define([]ast.Expr{data, err}, bst.Call("io", "ReadAll", rcvr.Expr))
			return Transform{
				Stmts:   []ast.Stmt{readStmt, panicOnErr(err, it)},
				Expr:    bst.Call(nil, "string", data),
				Imports: []string{"io"},
			}
		},
	})

	// Instance method: File#puts(str) → f.WriteString(str + "\n")
	FileType.Instance.Def("puts", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			str := bst.Binary(args[0].Expr, token.ADD, bst.String(`\n`))
			return Transform{
				Expr: bst.Call(rcvr.Expr, "WriteString", str),
			}
		},
	})

	// Instance method: File#path → f.Name()
	FileType.Instance.Def("path", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Name"),
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

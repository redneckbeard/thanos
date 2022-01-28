package types

import (
	"go/ast"

	"github.com/redneckbeard/thanos/bst"
)

type kernel struct {
	*proto
}

var (
	KernelType  = kernel{newProto("Kernel", "", ClassRegistry)}
	KernelClass = NewClass("Kernel", "", KernelType, ClassRegistry)
)

func (t kernel) Equals(t2 Type) bool { return t == t2 }
func (t kernel) String() string      { return "kernel" }
func (t kernel) GoType() string      { return "n/a" }
func (t kernel) IsComposite() bool   { return false }

func (t kernel) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return t.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t kernel) BlockArgTypes(m string, args []Type) []Type {
	return t.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t kernel) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return t.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t kernel) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t kernel) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t kernel) HasMethod(m string) bool {
	return t.proto.HasMethod(m, false)
}

func (t kernel) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func init() {
	KernelType.Def("puts", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			stmts := []ast.Stmt{}
			// `puts` inserts newlines after every argument, so we have one print function call for each here
			for _, arg := range args {
				// For any args that are interpolated strings, at this point we've
				// already translated them to the appropriate C-style interpolated
				// string. It would be weird in Go to call out to fmt.Sprintf for an
				// arg to fmt.Println, so here we grab those nodes, change the method
				// call to Printf, and insert a newline into the end of the string.
				if call, ok := arg.Expr.(*ast.CallExpr); ok {
					if fname, hasReceiver := call.Fun.(*ast.SelectorExpr); hasReceiver && fname.Sel.Name == "Sprintf" {
						fname.Sel = it.Get("Printf")
						lit := call.Args[0].(*ast.BasicLit)
						lit.Value = lit.Value[:len(lit.Value)-1] + `\n"`
						stmts = append(stmts, &ast.ExprStmt{X: call})
						continue
					}
				}
				stmts = append(stmts, &ast.ExprStmt{
					X: bst.Call("fmt", "Println", arg.Expr),
				})
			}
			return Transform{
				Stmts:   stmts,
				Imports: []string{"fmt"},
			}
		},
	})
	KernelType.Def("gauntlet", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{}
		},
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return NilType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: &ast.BadExpr{},
			}
		},
	})
	KernelType.Def("require", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{}
		},
	})
}

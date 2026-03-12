package types

import (
	"go/ast"
	"go/token"

	"github.com/redneckbeard/thanos/bst"
)

// MixinContext provides information from the parser to mixin Apply functions.
// For Enumerable, it carries a lazy resolver for the each method's element type.
type MixinContext map[string]interface{}

// Mixin represents a Ruby module that can be included in a class.
// RequiredMethods lists methods the class must define (e.g., <=> for Comparable).
// Apply defines the provided methods on the class instance.
type Mixin struct {
	Name            string
	RequiredMethods []string
	Apply           func(instance Instance, ctx MixinContext)
}

// MixinRegistry holds all known module mixins.
var MixinRegistry = map[string]*Mixin{}

func RegisterMixin(m *Mixin) {
	MixinRegistry[m.Name] = m
}

func init() {
	RegisterMixin(&Mixin{
		Name:            "Comparable",
		RequiredMethods: []string{"<=>"},
		Apply: func(instance Instance, ctx MixinContext) {
			spaceshipGoName := "Spaceship"

			// Check if the class defines <=> with a different Go name
			if spec, ok := instance.Resolve("<=>"); ok && spec.TransformAST != nil {
				_ = spec
			}

			// Collect user-defined method names to avoid overwriting them
			userDefined := map[string]bool{}
			if methods, ok := ctx["userMethods"]; ok {
				if m, ok := methods.(map[string]bool); ok {
					userDefined = m
				}
			}

			comparisons := []struct {
				op  string
				tok token.Token
			}{
				{"<", token.LSS},
				{">", token.GTR},
				{"<=", token.LEQ},
				{">=", token.GEQ},
				{"==", token.EQL},
			}

			for _, cmp := range comparisons {
				// Skip operators the class defines itself (user-defined takes priority)
				if userDefined[cmp.op] {
					continue
				}
				tok := cmp.tok
				goName := spaceshipGoName
				instance.Def(cmp.op, MethodSpec{
					ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
						return BoolType, nil
					},
					TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
						return Transform{
							Expr: bst.Binary(
								bst.Call(rcvr.Expr, goName, args[0].Expr),
								tok,
								bst.Int(0),
							),
						}
					},
				})
			}

			instance.Def("between?", MethodSpec{
				ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
					return BoolType, nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					goName := spaceshipGoName
					return Transform{
						Expr: bst.Binary(
							bst.Binary(bst.Call(rcvr.Expr, goName, args[0].Expr), token.GEQ, bst.Int(0)),
							token.LAND,
							bst.Binary(bst.Call(rcvr.Expr, goName, args[1].Expr), token.LEQ, bst.Int(0)),
						),
					}
				},
			})

			instance.Def("clamp", MethodSpec{
				ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
					return receiverType, nil
				},
				TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
					goName := spaceshipGoName
					result := it.New("clamped")
					stmts := []ast.Stmt{
						bst.Define(result, rcvr.Expr),
						&ast.IfStmt{
							Cond: bst.Binary(bst.Call(rcvr.Expr, goName, args[0].Expr), token.LSS, bst.Int(0)),
							Body: &ast.BlockStmt{List: []ast.Stmt{bst.Assign(result, args[0].Expr)}},
						},
						&ast.IfStmt{
							Cond: bst.Binary(bst.Call(rcvr.Expr, goName, args[1].Expr), token.GTR, bst.Int(0)),
							Body: &ast.BlockStmt{List: []ast.Stmt{bst.Assign(result, args[1].Expr)}},
						},
					}
					return Transform{
						Expr:  result,
						Stmts: stmts,
					}
				},
			})
		},
	})
}

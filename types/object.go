package types

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/redneckbeard/thanos/bst"
)

type Object struct {
	*proto
}

var ObjectType = Object{newProto("Object", "", ClassRegistry)}

var ObjectClass = NewClass("Object", "", ObjectType, ClassRegistry)

func (t Object) Equals(t2 Type) bool { return t == t2 }
func (t Object) String() string      { return "Object" }
func (t Object) GoType() string      { return "" }
func (t Object) IsComposite() bool   { return false }

func (t Object) MethodReturnType(m string, b Type, args []Type) (Type, error) {
	return ObjectType.proto.MustResolve(m, false).ReturnType(t, b, args)
}

func (t Object) BlockArgTypes(m string, args []Type) []Type {
	return ObjectType.proto.MustResolve(m, false).blockArgs(t, args)
}

func (t Object) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	return ObjectType.proto.MustResolve(m, false).TransformAST(TypeExpr{t, rcvr}, args, blk, it)
}

func (t Object) Resolve(m string) (MethodSpec, bool) {
	return t.proto.Resolve(m, false)
}

func (t Object) MustResolve(m string) MethodSpec {
	return t.proto.MustResolve(m, false)
}

func (t Object) HasMethod(m string) bool {
	return ObjectType.proto.HasMethod(m, false)
}

func (t Object) Alias(existingMethod, newMethod string) {
	t.proto.MakeAlias(existingMethod, newMethod, false)
}

func logicalOperatorSpec(tok token.Token) MethodSpec {
	return MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			left, right := rcvr.Expr, args[0].Expr
			//TODO in both these cases, this will be wrong for numeric types and strings
			if rcvr.Type != BoolType {
				left = bst.Binary(left, token.NEQ, it.Get("nil"))
			}
			if args[0].Type != BoolType {
				right = bst.Binary(right, token.NEQ, it.Get("nil"))
			}
			return Transform{
				Expr: bst.Binary(left, tok, right),
			}
		},
	}
}

func init() {
	ObjectType.Def("==", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("reflect", "DeepEqual", rcvr.Expr, args[0].Expr),
				Imports: []string{"reflect"},
			}
		},
	})

	ObjectType.Def("&&", logicalOperatorSpec(token.LAND))
	ObjectType.Def("||", logicalOperatorSpec(token.LOR))

	ObjectType.Def("is_a?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		// skip all iteration in target source and just expand
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var isA bool
			targetClass := args[0].Type
			class, _ := ClassRegistry.Get(rcvr.Type.ClassName())
			for class != nil {
				if class == targetClass {
					isA = true
					break
				}
				class = class.parent
			}
			return Transform{
				Expr: it.Get(strconv.FormatBool(isA)),
			}
		},
	})
	ObjectType.Def("instance_of?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		// skip all iteration in target source and just expand
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			var isInstance bool
			targetClass := args[0].Type
			if class, ok := targetClass.(*Class); ok {
				if rcvr.Type == class.Instance.(Type) {
					isInstance = true
				}
			}
			return Transform{
				Expr: it.Get(strconv.FormatBool(isInstance)),
			}
		},
	})
	ObjectClass.Def("instance_methods", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return NewArray(SymbolType), nil
		},
		// skip all iteration in target source and just expand
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			instanceType := rcvr.Type.(*Class).Instance.(Type)
			return instanceType.TransformAST("methods", rcvr.Expr, args, blk, it)
		},
	})

	ObjectType.Def("methods", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return NewArray(SymbolType), nil
		},
		// skip all iteration in target source and just expand
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			names := []ast.Expr{}
			class, err := ClassRegistry.Get(rcvr.Type.ClassName())
			if err != nil {
				panic(err)
			}
			methods := map[string]bool{}
			for class != nil {
				for k := range class.Instance.Methods() {
					methods[k] = true
				}
				class = class.parent
			}
			for k := range methods {
				names = append(names, bst.String(k))
			}
			return Transform{
				Expr: &ast.CompositeLit{
					Type: &ast.ArrayType{
						Elt: it.Get("string"),
					},
					Elts: names,
				},
			}
		},
	})

	ObjectType.Def("to_json", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return StringType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr:    bst.Call("shims", "JSONGenerate", rcvr.Expr),
				Imports: []string{"github.com/redneckbeard/thanos/shims"},
			}
		},
	})

	// Deprecated and unsupported methods, or general uselessness
	noops := []string{"taint", "untaint", "trust", "untrust", "itself", "freeze"}
	for _, noop := range noops {
		ObjectType.Def(noop, NoopReturnSelf)
	}

	ObjectType.Def("tainted?", AlwaysFalse)
	ObjectType.Def("untrusted?", AlwaysFalse)
	ObjectType.Def("frozen?", AlwaysTrue)

	ObjectType.Def("tap", MethodSpec{
		blockArgs: func(r Type, args []Type) []Type {
			return []Type{r}
		},
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return receiverType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			tapVar := blk.Args[0]
			init := bst.Define(tapVar, rcvr.Expr)
			StripBlockReturn(blk)
			stmts := []ast.Stmt{init}
			stmts = append(stmts, blk.Statements...)
			return Transform{
				Expr:  tapVar,
				Stmts: stmts,
			}
		},
	})

	ObjectType.Def("respond_to?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// At compile time, check if the receiver type has the named method
			if len(args) > 0 {
				methodName := ""
				// The argument is a symbol like :foo — extract the name
				if lit, ok := args[0].Expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					methodName = strings.Trim(lit.Value, `"`)
				}
				if methodName != "" {
					if rcvr.Type.HasMethod(methodName) {
						return Transform{Expr: it.Get("true")}
					}
					return Transform{Expr: it.Get("false")}
				}
			}
			// Fallback: can't determine at compile time
			return Transform{Expr: it.Get("false")}
		},
	})

	ObjectType.Def("nil?", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if _, ok := rcvr.Type.(Optional); ok {
				return Transform{
					Expr: bst.Binary(rcvr.Expr, token.EQL, it.Get("nil")),
				}
			}
			return Transform{
				Expr: it.Get("false"),
			}
		},
	})
}

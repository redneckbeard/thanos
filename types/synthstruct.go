package types

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/redneckbeard/thanos/bst"
)

// SynthField describes one field in a synthesized Go struct.
type SynthField struct {
	Name string // "Field0", "Field1", etc.
	Type Type
}

// SynthStruct is a synthesized Go struct type created when Ruby code uses
// heterogeneous array literals (Tuples) as elements of a homogeneous array.
// For example, `links[k] = [prev, i, j]` produces a struct with three fields.
type SynthStruct struct {
	Name   string       // "LinksEntry"
	Fields []SynthField
}

func NewSynthStruct(name string, fields []SynthField) *SynthStruct {
	return &SynthStruct{Name: name, Fields: fields}
}

func (s *SynthStruct) GoType() string    { return "*" + s.Name }
func (s *SynthStruct) String() string    { return s.GoType() }
func (s *SynthStruct) ClassName() string { return s.Name }
func (s *SynthStruct) IsComposite() bool { return false }
func (s *SynthStruct) IsMultiple() bool  { return false }

func (s *SynthStruct) Equals(t2 Type) bool {
	if ss, ok := t2.(*SynthStruct); ok {
		return s.Name == ss.Name
	}
	return false
}

func (s *SynthStruct) HasMethod(m string) bool {
	switch m {
	case "[]", "[]=", "nil?":
		return true
	}
	return false
}

func (s *SynthStruct) MethodReturnType(m string, blockRet Type, args []Type) (Type, error) {
	switch m {
	case "[]":
		if len(args) > 0 {
			if idx, ok := constIntType(args[0]); ok && idx >= 0 && idx < len(s.Fields) {
				return s.Fields[idx].Type, nil
			}
		}
		return AnyType, nil
	case "[]=":
		if len(args) > 1 {
			return args[1], nil
		}
		return AnyType, nil
	case "nil?":
		return BoolType, nil
	}
	return nil, fmt.Errorf("SynthStruct %s does not support method '%s'", s.Name, m)
}

func (s *SynthStruct) GetMethodSpec(m string) (MethodSpec, bool) {
	switch m {
	case "[]":
		return s.bracketAccessSpec(), true
	case "[]=":
		return s.bracketAssignSpec(), true
	case "nil?":
		return s.nilCheckSpec(), true
	}
	return MethodSpec{}, false
}

func (s *SynthStruct) BlockArgTypes(m string, args []Type) []Type {
	return nil
}

func (s *SynthStruct) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	if spec, ok := s.GetMethodSpec(m); ok && spec.TransformAST != nil {
		return spec.TransformAST(TypeExpr{s, rcvr}, args, blk, it)
	}
	return Transform{}
}

func (s *SynthStruct) bracketAccessSpec() MethodSpec {
	return MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return s.MethodReturnType("[]", blockReturnType, args)
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if idx, ok := constIntExpr(args[0].Expr); ok && idx >= 0 && idx < len(s.Fields) {
				return Transform{
					Expr: &ast.SelectorExpr{
						X:   rcvr.Expr,
						Sel: it.Get(s.Fields[idx].Name),
					},
				}
			}
			return Transform{
				Expr: bst.Call(rcvr.Expr, "Get", args[0].Expr),
			}
		},
	}
}

func (s *SynthStruct) bracketAssignSpec() MethodSpec {
	return MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			if len(args) > 1 {
				return args[1], nil
			}
			return AnyType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			// args[0] = index, args[1] = value
			if idx, ok := constIntExpr(args[0].Expr); ok && idx >= 0 && idx < len(s.Fields) {
				return Transform{
					Stmts: []ast.Stmt{
						bst.Assign(
							&ast.SelectorExpr{
								X:   rcvr.Expr,
								Sel: it.Get(s.Fields[idx].Name),
							},
							args[1].Expr,
						),
					},
				}
			}
			return Transform{
				Stmts: []ast.Stmt{
					&ast.ExprStmt{
						X: bst.Call(rcvr.Expr, "Set", args[0].Expr, args[1].Expr),
					},
				},
			}
		},
	}
}

func (s *SynthStruct) nilCheckSpec() MethodSpec {
	return MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			return Transform{
				Expr: bst.Binary(rcvr.Expr, token.EQL, it.Get("nil")),
			}
		},
	}
}

// constIntType checks if a Type represents a known constant integer.
// This is used during type inference where we only have Type info, not AST nodes.
func constIntType(t Type) (int, bool) {
	// During type inference we can't extract the constant value from a Type alone.
	// The actual constant extraction happens at the AST level via constIntExpr.
	return 0, false
}

// constIntExpr extracts a constant integer value from an AST expression.
func constIntExpr(expr ast.Expr) (int, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.INT {
			val, err := strconv.Atoi(e.Value)
			if err == nil {
				return val, true
			}
		}
	case *ast.Ident:
		// Check for identifiers that represent integer literals
		val, err := strconv.Atoi(e.Name)
		if err == nil {
			return val, true
		}
	}
	return 0, false
}

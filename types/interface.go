package types

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/redneckbeard/thanos/bst"
)

// DuckInterface represents a synthesized Go interface type created when
// multiple concrete types are passed to the same method parameter.
// All ConcreteTypes must implement all methods listed in Methods.
type DuckInterface struct {
	Name          string   // e.g., "ProcessCallbacksIface"
	MethodNames   []string // ordered list of required method names
	ConcreteTypes []Type   // T1, T2, ... that all satisfy this interface
}

func (t *DuckInterface) GoType() string    { return t.Name }
func (t *DuckInterface) ClassName() string { return t.Name }
func (t *DuckInterface) String() string    { return t.Name }
func (t *DuckInterface) IsComposite() bool { return false }
func (t *DuckInterface) IsMultiple() bool  { return false }

func (t *DuckInterface) Equals(t2 Type) bool {
	if d, ok := t2.(*DuckInterface); ok {
		return t.Name == d.Name
	}
	return false
}

func (t *DuckInterface) HasMethod(m string) bool {
	if t.hasInterfaceMethod(m) {
		return true
	}
	// respond_to? is always available (inherited from Object)
	if m == "respond_to?" {
		return true
	}
	// Optional methods (available on some concrete types) are callable
	// via type assertion dispatch
	return t.HasOptionalMethod(m)
}

// HasOptionalMethod checks if any concrete type has the method even though
// it's not required by the interface (not all concrete types have it).
func (t *DuckInterface) HasOptionalMethod(m string) bool {
	if t.hasInterfaceMethod(m) {
		return false // it's a required method, not optional
	}
	for _, ct := range t.ConcreteTypes {
		if ct.HasMethod(m) {
			return true
		}
	}
	return false
}

// hasInterfaceMethod checks only the interface's own method list.
func (t *DuckInterface) hasInterfaceMethod(m string) bool {
	for _, name := range t.MethodNames {
		if name == m {
			return true
		}
	}
	return false
}

func (t *DuckInterface) MethodReturnType(m string, blockRet Type, args []Type) (Type, error) {
	if m == "respond_to?" {
		return BoolType, nil
	}
	// For interface methods, delegate to the first concrete type
	if t.hasInterfaceMethod(m) {
		return t.ConcreteTypes[0].MethodReturnType(m, blockRet, args)
	}
	// For optional methods, find the first concrete type that has it
	for _, ct := range t.ConcreteTypes {
		if ct.HasMethod(m) {
			return ct.MethodReturnType(m, blockRet, args)
		}
	}
	return nil, fmt.Errorf("no concrete type for %s has method '%s'", t.Name, m)
}

func (t *DuckInterface) GetMethodSpec(m string) (MethodSpec, bool) {
	if m == "respond_to?" {
		return t.respondToSpec(), true
	}
	// Interface methods: plain method call (Go interface dispatch)
	if t.hasInterfaceMethod(m) {
		origSpec, _ := t.ConcreteTypes[0].GetMethodSpec(m)
		return MethodSpec{
			ReturnType: origSpec.ReturnType,
			TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
				goName := duckGoName(m)
				argExprs := UnwrapTypeExprs(args)
				if blk != nil {
					argExprs = append(argExprs, blk.FuncLit(it))
				}
				return Transform{
					Expr: bst.Call(rcvr.Expr, goName, argExprs...),
				}
			},
		}, true
	}
	// Optional methods: type assertion dispatch
	if t.HasOptionalMethod(m) {
		// Find first concrete type with this method
		var origSpec MethodSpec
		for _, ct := range t.ConcreteTypes {
			if ct.HasMethod(m) {
				origSpec, _ = ct.GetMethodSpec(m)
				break
			}
		}
		return MethodSpec{
			ReturnType: origSpec.ReturnType,
			TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
				goName := duckGoName(m)
				argExprs := UnwrapTypeExprs(args)
				if blk != nil {
					argExprs = append(argExprs, blk.FuncLit(it))
				}
				// Emit: rcvr.(interface{ Method(...) ... }).Method(args)
				if BuildAnonInterface != nil {
					anonIface := BuildAnonInterface(rcvr.Type.(*DuckInterface), m)
					if anonIface != nil {
						asserted := &ast.TypeAssertExpr{
							X:    rcvr.Expr,
							Type: anonIface,
						}
						return Transform{
							Expr: bst.Call(asserted, goName, argExprs...),
						}
					}
				}
				// Fallback: plain call (may not compile)
				return Transform{
					Expr: bst.Call(rcvr.Expr, goName, argExprs...),
				}
			},
		}, true
	}
	return MethodSpec{}, false
}

func (t *DuckInterface) BlockArgTypes(m string, args []Type) []Type {
	return t.ConcreteTypes[0].BlockArgTypes(m, args)
}

func (t *DuckInterface) TransformAST(m string, rcvr ast.Expr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
	// Check for a custom spec (respond_to?, optional methods)
	if spec, ok := t.GetMethodSpec(m); ok && spec.TransformAST != nil {
		return spec.TransformAST(TypeExpr{t, rcvr}, args, blk, it)
	}
	goName := duckGoName(m)
	argExprs := UnwrapTypeExprs(args)
	if blk != nil {
		argExprs = append(argExprs, blk.FuncLit(it))
	}
	return Transform{
		Expr: bst.Call(rcvr, goName, argExprs...),
	}
}

// Satisfies checks whether a candidate type has all the required methods.
func (t *DuckInterface) Satisfies(candidate Type) bool {
	for _, m := range t.MethodNames {
		if !candidate.HasMethod(m) {
			return false
		}
	}
	return true
}

// respondToSpec returns a MethodSpec for respond_to? on a DuckInterface.
// For methods in the interface: compile-time true.
// For optional methods (on some but not all concrete types): runtime type assertion.
// For unknown methods: compile-time false.
func (t *DuckInterface) respondToSpec() MethodSpec {
	return MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return BoolType, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
			if len(args) == 0 {
				return Transform{Expr: it.Get("false")}
			}
			methodName := ""
			if lit, ok := args[0].Expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				methodName = strings.Trim(lit.Value, `"`)
			}
			if methodName == "" {
				return Transform{Expr: it.Get("false")}
			}

			// Method is in the interface — compile-time true
			if t.hasInterfaceMethod(methodName) {
				return Transform{Expr: it.Get("true")}
			}

			// Check if any concrete type has the method
			if !t.HasOptionalMethod(methodName) {
				return Transform{Expr: it.Get("false")}
			}

			// Runtime type assertion check.
			// Build: _, ok := rcvr.(interface{ MethodName(...) ... })
			if BuildAnonInterface != nil {
				anonIface := BuildAnonInterface(t, methodName)
				if anonIface != nil {
					okIdent := it.New("ok")
					stmt := bst.Define(
						[]ast.Expr{it.Get("_"), okIdent},
						&ast.TypeAssertExpr{
							X:    rcvr.Expr,
							Type: anonIface,
						},
					)
					return Transform{
						Stmts: []ast.Stmt{stmt},
						Expr:  okIdent,
					}
				}
			}

			return Transform{Expr: it.Get("false")}
		},
	}
}

// BuildAnonInterface is set by the parser package to build anonymous interface
// types for optional method type assertions. It looks up the method signature
// on the concrete type via classMethodSets.
var BuildAnonInterface func(iface *DuckInterface, methodName string) *ast.InterfaceType

// GoMethodName is a function variable set by the parser package to convert
// Ruby method names to Go names. This avoids a circular import.
var GoMethodName func(string) string

// duckGoName converts a Ruby method name to Go style for interface methods.
func duckGoName(rubyName string) string {
	if GoMethodName != nil {
		return GoMethodName(rubyName)
	}
	// Fallback: simple title-case
	name := strings.TrimRight(rubyName, "!")
	name = strings.Title(name)
	if strings.HasSuffix(name, "?") {
		name = "Is" + strings.TrimRight(name, "?")
	}
	if strings.HasSuffix(name, "=") {
		name = "Set" + strings.TrimRight(name, "=")
	}
	return name
}

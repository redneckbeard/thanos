package parser

import (
	"go/ast"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

func init() {
	// Wire up the GoMethodName function so types/interface.go can convert
	// Ruby method names to Go without importing parser.
	types.GoMethodName = GoName

	// Wire up BuildAnonInterface so DuckInterface.respondToSpec can build
	// anonymous interface types for optional method type assertions.
	types.BuildAnonInterface = buildAnonInterface
}

// DuckInterfaces stores synthesized interfaces for the compiler to emit.
// Populated during AnalyzeArguments when type conflicts are resolved via
// duck-type inference.
var DuckInterfaces []*types.DuckInterface

// isRespondToGuard checks if a condition node is `paramName.respond_to?(:something)`
func isRespondToGuard(cond Node, paramName string) bool {
	mc, ok := cond.(*MethodCall)
	if !ok || mc.MethodName != "respond_to?" {
		return false
	}
	ident, ok := mc.Receiver.(*IdentNode)
	return ok && ident.Val == paramName
}

// findMethodCallsOnParam walks the AST recursively and returns all method
// names called on an identifier matching paramName. Calls guarded by
// respond_to? checks on the same param are excluded (they're optional).
func findMethodCallsOnParam(stmts Statements, paramName string) []string {
	seen := map[string]bool{}
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil {
			return
		}
		switch v := n.(type) {
		case *MethodCall:
			if ident, ok := v.Receiver.(*IdentNode); ok && ident.Val == paramName {
				seen[v.MethodName] = true
			}
			walk(v.Receiver)
			for _, a := range v.Args {
				walk(a)
			}
			if v.Block != nil {
				walkStmts(v.Block.Body.Statements, walk)
			}
		case *Condition:
			// Skip the true branch of respond_to? guards — those are optional methods
			if isRespondToGuard(v.Condition, paramName) {
				// Still walk the condition itself (for the respond_to? call)
				// but skip the guarded body
				if v.False != nil {
					walk(v.False)
				}
			} else {
				walk(v.Condition)
				walkStmts(v.True, walk)
				if v.False != nil {
					walk(v.False)
				}
			}
		case *CaseNode:
			walk(v.Value)
			for _, w := range v.Whens {
				walkStmts(w.Statements, walk)
			}
		case *WhileNode:
			walk(v.Condition)
			walkStmts(v.Body, walk)
		case *ForInNode:
			walk(v.In)
			walkStmts(v.Body, walk)
		case *BeginNode:
			walkStmts(v.Body, walk)
			for _, clause := range v.RescueClauses {
				walkStmts(clause.Body, walk)
			}
			walkStmts(v.EnsureBody, walk)
		case *ReturnNode:
			for _, val := range v.Val {
				walk(val)
			}
		case *AssignmentNode:
			for _, r := range v.Right {
				walk(r)
			}
		case *InfixExpressionNode:
			walk(v.Left)
			walk(v.Right)
		case *NotExpressionNode:
			walk(v.Arg)
		case Statements:
			walkStmts(v, walk)
		}
	}
	for _, s := range stmts {
		walk(s)
	}
	result := make([]string, 0, len(seen))
	for m := range seen {
		result = append(result, m)
	}
	return result
}

func walkStmts(stmts []Node, walk func(Node)) {
	for _, s := range stmts {
		walk(s)
	}
}

// tryBuildDuckInterface attempts to build a DuckInterface when two different
// Instance types are passed to the same parameter. Returns nil if the types
// are incompatible (not both user-defined, or missing required methods).
func tryBuildDuckInterface(method *Method, param *Param, existingType, newType types.Type) *types.DuckInterface {
	// If existing type is already a DuckInterface, check if newType satisfies it
	if existing, ok := existingType.(*types.DuckInterface); ok {
		if existing.Satisfies(newType) {
			// Add to concrete types if not already present
			for _, ct := range existing.ConcreteTypes {
				if ct.Equals(newType) {
					return existing
				}
			}
			existing.ConcreteTypes = append(existing.ConcreteTypes, newType)
			return existing
		}
		return nil
	}

	// Both types must be Instance (user-defined classes)
	_, ok1 := existingType.(types.Instance)
	_, ok2 := newType.(types.Instance)
	if !ok1 || !ok2 {
		return nil
	}

	// Find what methods are called on this param in the method body
	requiredMethods := findMethodCallsOnParam(method.Body.Statements, param.Name)
	if len(requiredMethods) == 0 {
		return nil
	}

	// Verify both types have all required methods
	for _, m := range requiredMethods {
		if !existingType.HasMethod(m) || !newType.HasMethod(m) {
			return nil
		}
	}

	// Build the interface name: MethodNameParamNameIface
	ifaceName := strings.Title(method.Name) + strings.Title(param.Name) + "Iface"

	iface := &types.DuckInterface{
		Name:          ifaceName,
		MethodNames:   requiredMethods,
		ConcreteTypes: []types.Type{existingType, newType},
	}

	return iface
}

// BuildInterfaceMethodSignatures builds the Go interface method list from
// a DuckInterface by examining the analyzed methods on the first concrete type.
func BuildInterfaceMethodSignatures(iface *types.DuckInterface) []InterfaceMethodSig {
	var sigs []InterfaceMethodSig

	concrete := iface.ConcreteTypes[0]
	ms, ok := classMethodSets[concrete]
	if !ok {
		return nil
	}

	for _, methodName := range iface.MethodNames {
		m, exists := ms.Methods[methodName]
		if !exists {
			continue
		}
		sig := InterfaceMethodSig{
			RubyName: methodName,
			GoName:   m.GoName(),
			Params:   m.Params,
			RetType:  m.ReturnType(),
		}
		if m.Block != nil {
			sig.Block = m.Block
		}
		sigs = append(sigs, sig)
	}

	return sigs
}

// InterfaceMethodSig holds the info needed to emit one method in a Go interface.
type InterfaceMethodSig struct {
	RubyName string
	GoName   string
	Params   []*Param
	RetType  types.Type
	Block    *BlockParam
}

// buildAnonInterface creates an anonymous interface type AST for a single
// method, used in type assertions for optional respond_to? checks.
// Looks up the method on the first concrete type that has it.
func buildAnonInterface(iface *types.DuckInterface, methodName string) *ast.InterfaceType {
	// Find a concrete type that has this method
	for _, ct := range iface.ConcreteTypes {
		if !ct.HasMethod(methodName) {
			continue
		}
		ms, ok := classMethodSets[ct]
		if !ok {
			continue
		}
		m, exists := ms.Methods[methodName]
		if !exists {
			continue
		}

		// Build param list
		var params []*ast.Field
		for _, p := range m.Params {
			pType := "interface{}"
			if p.Type() != nil {
				pType = p.Type().GoType()
			}
			params = append(params, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(p.Name)},
				Type:  ast.NewIdent(pType),
			})
		}

		// Build return type
		var results *ast.FieldList
		if m.ReturnType() != nil && m.ReturnType() != types.NilType {
			results = &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent(m.ReturnType().GoType())},
				},
			}
		}

		return &ast.InterfaceType{
			Methods: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent(m.GoName())},
						Type: &ast.FuncType{
							Params: &ast.FieldList{List: params},
							Results: results,
						},
					},
				},
			},
		}
	}
	return nil
}

// ResetDuckInterfaces clears the global list (for test isolation).
func ResetDuckInterfaces() {
	DuckInterfaces = nil
}

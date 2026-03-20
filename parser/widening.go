package parser

import (
	"fmt"
	"os"

	"github.com/redneckbeard/thanos/types"
)

// typeWidening records a case where consumer usage implies a wider type
// than what the producer provides.
type typeWidening struct {
	varName    string       // the variable whose type was widened
	widerType  types.Type   // the wider type (after promotion)
	sourceCall *MethodCall  // the method call that produced the value
}

// propagateTypeWidenings runs after all analysis passes. It detects cases
// where consumer code (e.g., .nil? checks on array elements) has widened a
// variable's type beyond what the producing function returns, and propagates
// that wider type back to the producer.
//
// This implements backwards type propagation: thanos assumes the Ruby code
// is correct, so if consumer code checks .nil? on array elements, the
// producing function must be able to return nil elements.
func (r *Root) propagateTypeWidenings() {
	var widenings []typeWidening

	// Walk all module class methods
	for _, mod := range r.TopLevelModules {
		r.collectModuleWidenings(mod, &widenings)
	}
	// Walk all class methods
	for _, cls := range r.Classes {
		for _, m := range cls.MethodSet.Methods {
			r.collectMethodWidenings(m, &widenings)
		}
		for _, m := range cls.ClassMethods {
			r.collectMethodWidenings(m, &widenings)
		}
	}
	// Walk global methods
	for _, m := range r.MethodSetStack.Peek().Methods {
		r.collectMethodWidenings(m, &widenings)
	}

	// Apply widenings
	for _, w := range widenings {
		r.applyWidening(w)
	}
}

func (r *Root) collectModuleWidenings(mod *Module, widenings *[]typeWidening) {
	for _, m := range mod.ClassMethods {
		r.collectMethodWidenings(m, widenings)
	}
	for _, m := range mod.MethodSet.Methods {
		r.collectMethodWidenings(m, widenings)
	}
	for _, sub := range mod.Modules {
		r.collectModuleWidenings(sub, widenings)
	}
	for _, cls := range mod.Classes {
		for _, m := range cls.MethodSet.Methods {
			r.collectMethodWidenings(m, widenings)
		}
		for _, m := range cls.ClassMethods {
			r.collectMethodWidenings(m, widenings)
		}
	}
}

// collectMethodWidenings walks a method body doing AST-level pattern matching
// to find cases where a variable's elements are nil-checked but the variable
// was assigned from a function returning non-Optional elements.
//
// This works even when gem method body analysis fails partway through,
// because it examines the AST directly rather than relying on scope types.
func (r *Root) collectMethodWidenings(m *Method, widenings *[]typeWidening) {
	if m.Body == nil {
		return
	}

	// Phase 1: Map variable names to the MethodCall that produced them
	assignments := map[string]*MethodCall{}
	for _, stmt := range m.Body.Statements {
		collectAssignments(stmt, assignments)
	}

	// Phase 2: Find .nil? calls on bracket access of those variables
	nilCheckedVars := map[string]bool{}
	for _, stmt := range m.Body.Statements {
		findNilCheckedArrayVars(stmt, nilCheckedVars)
	}


	// Phase 3: For each nil-checked variable that was assigned from a method call
	// returning a non-Optional array, record the widening
	for varName := range nilCheckedVars {
		call, ok := assignments[varName]
		if !ok {
			continue
		}
		callRetType := call.Type()
		if callRetType == nil {
			continue
		}
		arr, isArray := callRetType.(types.Array)
		if !isArray {
			continue
		}
		if _, alreadyOpt := arr.Element.(types.Optional); alreadyOpt {
			continue
		}
		widerType := types.NewArray(types.NewOptional(arr.Element))
		*widenings = append(*widenings, typeWidening{
			varName:    varName,
			widerType:  widerType,
			sourceCall: call,
		})
	}
}

// collectAssignments walks an AST node and records variable-to-MethodCall
// assignments (var = method_call(...)).
func collectAssignments(node Node, assignments map[string]*MethodCall) {
	switch n := node.(type) {
	case *AssignmentNode:
		if len(n.Left) == 1 && len(n.Right) == 1 {
			if ident, ok := n.Left[0].(*IdentNode); ok {
				if call, ok := n.Right[0].(*MethodCall); ok {
					assignments[ident.Val] = call
				}
			}
		}
	case *MethodCall:
		if n.Block != nil && n.Block.Body != nil {
			for _, stmt := range n.Block.Body.Statements {
				collectAssignments(stmt, assignments)
			}
		}
	case *WhileNode:
		for _, stmt := range n.Body {
			collectAssignments(stmt, assignments)
		}
	case *Condition:
		for _, stmt := range n.True {
			collectAssignments(stmt, assignments)
		}
		if n.False != nil {
			collectAssignments(n.False, assignments)
		}
	}
}

// findNilCheckedArrayVars walks an AST node looking for the pattern
// var[idx].nil?() and records the variable names.
func findNilCheckedArrayVars(node Node, result map[string]bool) {
	switch n := node.(type) {
	case *MethodCall:
		// Check for var[idx].nil?
		if n.MethodName == "nil?" && n.Receiver != nil {
			if ba, ok := n.Receiver.(*BracketAccessNode); ok {
				if ident, ok := ba.Composite.(*IdentNode); ok {
					result[ident.Val] = true
				}
			}
		}
		// Recurse into args
		for _, arg := range n.Args {
			findNilCheckedArrayVars(arg, result)
		}
		// Recurse into block
		if n.Block != nil && n.Block.Body != nil {
			for _, stmt := range n.Block.Body.Statements {
				findNilCheckedArrayVars(stmt, result)
			}
		}
	case *Condition:
		findNilCheckedArrayVars(n.Condition, result)
		for _, stmt := range n.True {
			findNilCheckedArrayVars(stmt, result)
		}
		if n.False != nil {
			findNilCheckedArrayVars(n.False, result)
		}
	case *WhileNode:
		for _, stmt := range n.Body {
			findNilCheckedArrayVars(stmt, result)
		}
	case *AssignmentNode:
		for _, r := range n.Right {
			findNilCheckedArrayVars(r, result)
		}
	case *ReturnNode:
		for _, v := range n.Val {
			findNilCheckedArrayVars(v, result)
		}
	case *NextNode:
		if n.Val != nil {
			findNilCheckedArrayVars(n.Val, result)
		}
	case *InfixExpressionNode:
		findNilCheckedArrayVars(n.Left, result)
		findNilCheckedArrayVars(n.Right, result)
	}
}

// isWiderArrayType returns true if wider is an array with Optional elements
// where narrower has non-Optional elements of the same base type.
func isWiderArrayType(wider, narrower types.Type) bool {
	wArr, wOk := wider.(types.Array)
	nArr, nOk := narrower.(types.Array)
	if !wOk || !nOk {
		return false
	}
	wOpt, wIsOpt := wArr.Element.(types.Optional)
	_, nIsOpt := nArr.Element.(types.Optional)
	if !wIsOpt || nIsOpt {
		return false
	}
	// The base types must match
	return wOpt.Element == nArr.Element
}

// applyWidening propagates a type widening back to the producing function.
func (r *Root) applyWidening(w typeWidening) {
	// Resolve the producing method from the call
	method := r.resolveMethodFromCall(w.sourceCall)
	if method == nil {
		return
	}

	// Emit warning when user code widens gem/library code
	if method.FromGem {
		fmt.Fprintf(os.Stderr, "note: widening return type of %s from %s to %s (consumer checks .nil? on elements)\n",
			method.Name, method.Body.ReturnType, w.widerType)
	}

	// Update the method's return type
	method.Body.ReturnType = w.widerType

	// Find the variable in the method body that is returned and promote it.
	r.promoteReturnedVariable(method, w.widerType)
}

// resolveMethodFromCall finds the Method node that a MethodCall invokes.
func (r *Root) resolveMethodFromCall(call *MethodCall) *Method {
	if call.Receiver == nil {
		// Bare function call — check global method set
		if m, ok := r.MethodSetStack.Peek().Methods[call.MethodName]; ok {
			return m
		}
		return nil
	}

	receiverType := call.Receiver.Type()
	if receiverType == nil {
		return nil
	}

	// Check classMethodSets for the receiver type (instance methods)
	if ms, ok := classMethodSets[receiverType]; ok {
		if m, ok := ms.Methods[call.MethodName]; ok {
			return m
		}
	}

	// For module/class method calls, the receiver is the class itself.
	// Check ClassMethods on modules.
	for _, mod := range r.TopLevelModules {
		if m := r.findModuleClassMethod(mod, receiverType, call.MethodName); m != nil {
			return m
		}
	}
	for _, cls := range r.Classes {
		for _, m := range cls.ClassMethods {
			if m.Name == call.MethodName {
				return m
			}
		}
	}

	return nil
}

// findModuleClassMethod recursively searches modules for a class method
// matching the given receiver type and method name.
func (r *Root) findModuleClassMethod(mod *Module, receiverType types.Type, methodName string) *Method {
	if modType := mod.Type(); modType != nil {
		if modType == receiverType {
			for _, m := range mod.ClassMethods {
				if m.Name == methodName {
					return m
				}
			}
		}
	}
	for _, sub := range mod.Modules {
		if m := r.findModuleClassMethod(sub, receiverType, methodName); m != nil {
			return m
		}
	}
	for _, cls := range mod.Classes {
		for _, m := range cls.ClassMethods {
			if m.Name == methodName {
				clsType := cls.Type()
				if clsType == receiverType {
					return m
				}
			}
		}
	}
	return nil
}

// promoteReturnedVariable finds the variable that a method returns and
// promotes its type in the method's scope to match the widened return type.
func (r *Root) promoteReturnedVariable(method *Method, widerType types.Type) {
	if method.Body == nil || len(method.Body.Statements) == 0 {
		return
	}

	// The return value is either the last expression (implicit return)
	// or an explicit ReturnNode. Extract the returned ident.
	last := method.Body.Statements[len(method.Body.Statements)-1]
	var ident *IdentNode
	switch n := last.(type) {
	case *IdentNode:
		ident = n
	case *ReturnNode:
		if len(n.Val) == 1 {
			if id, ok := n.Val[0].(*IdentNode); ok {
				ident = id
			}
		}
	}
	if ident == nil {
		return
	}

	// Promote the variable in the method's scope
	if method.Scope != nil {
		method.Scope.RefineVariableType(ident.Val, widerType)
	}

	// Also update the ident node's cached type
	ident.SetType(widerType)

	// Walk all bracket assignment statements to this variable and update
	// the composite node's type so the compiler emits correct code.
	for _, stmt := range method.Body.Statements {
		r.promoteBracketAssignments(stmt, ident.Val, widerType)
	}
}

// promoteBracketAssignments walks statements and updates the composite type
// on bracket assignments targeting the named variable.
func (r *Root) promoteBracketAssignments(node Node, varName string, widerType types.Type) {
	switch n := node.(type) {
	case *AssignmentNode:
		if len(n.Left) == 1 {
			if ba, ok := n.Left[0].(*BracketAssignmentNode); ok {
				if ident, ok := ba.Composite.(*IdentNode); ok && ident.Val == varName {
					ident.SetType(widerType)
				}
			}
		}
	case *WhileNode:
		for _, stmt := range n.Body {
			r.promoteBracketAssignments(stmt, varName, widerType)
		}
	case *Condition:
		for _, stmt := range n.True {
			r.promoteBracketAssignments(stmt, varName, widerType)
		}
		if n.False != nil {
			r.promoteBracketAssignments(n.False, varName, widerType)
		}
	case *MethodCall:
		if n.Block != nil && n.Block.Body != nil {
			for _, stmt := range n.Block.Body.Statements {
				r.promoteBracketAssignments(stmt, varName, widerType)
			}
		}
	}
}


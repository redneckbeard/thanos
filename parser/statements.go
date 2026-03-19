package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type ArgsNode []Node

func (n ArgsNode) String() string {
	return stdlib.Join[Node](n, ", ")
}

// Wrong but dummy for satisfying interface
func (n ArgsNode) Type() types.Type     { return n[0].Type() }
func (n ArgsNode) SetType(t types.Type) {}
func (n ArgsNode) LineNo() int          { return 0 }
func (n ArgsNode) File() string         { return "" }

func (n ArgsNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	panic("ArgsNode#TargetType should never be called")
}

func (n ArgsNode) Copy() Node {
	var copy []Node
	for _, arg := range n {
		copy = append(copy, arg.Copy())
	}
	return ArgsNode(copy)
}

func (n ArgsNode) FindByName(name string) (Node, error) {
	for _, arg := range n {
		if kv, ok := arg.(*KeyValuePair); ok && kv.Label == name {
			return kv, nil
		}
	}
	return nil, fmt.Errorf("No argument named '%s' found", name)
}

type ReturnNode struct {
	Val    ArgsNode
	_type  types.Type
	Pos
}

func (n *ReturnNode) String() string       { return fmt.Sprintf("(return %s)", n.Val) }
func (n *ReturnNode) Type() types.Type     { return n._type }
func (n *ReturnNode) SetType(t types.Type) { n._type = t }

func (n *ReturnNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if len(n.Val) == 1 {
		return GetType(n.Val[0], locals, class)
	}
	multiple := types.Multiple{}
	for _, single := range n.Val {
		t, err := GetType(single, locals, class)
		if err != nil {
			return t, err
		}
		multiple = append(multiple, t)
	}
	return multiple, nil
}

func (n *ReturnNode) Copy() Node {
	return &ReturnNode{n.Val.Copy().(ArgsNode), n._type, n.Pos}
}

type Statements []Node

func (stmts Statements) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	var lastReturnedType types.Type
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *AssignmentNode:
			if t, err := GetType(s, scope, class); err != nil {
				if !tolerantGetType {
					return nil, err
				}
			} else {
				lastReturnedType = t
			}
		case *Condition:
			// We need this to be semi-"live" since otherwise we can't surface an
			// error about a type mismatch between the branches. The type on the
			// condition will still be effectively memoized since it can just get the
			// cached value from the True side. Thus we call TargetType directly on
			// the node instead of going through GetType.
			if t, err := GetType(s, scope, class); err != nil {
				if !tolerantGetType {
					return nil, err
				}
			} else {
				lastReturnedType = t
			}
		case *IVarNode:
			if t, err := GetType(s, scope, class); err != nil {
				if !tolerantGetType {
					return nil, err
				}
			} else {
				lastReturnedType = t
			}
		default:
			if c, ok := stmt.(*MethodCall); ok {
				// Handle method chaining -- walk down to the first identifier or
				// literal and infer types on the way back up so that receiver type is
				// known for each subsequent method call
				chain := []Node{c}
				r := c.Receiver
				walking := true
				for walking {
					switch c := r.(type) {
					case *MethodCall:
						chain = append(chain, c)
						r = c.Receiver
					default:
						if c != nil {
							chain = append(chain, c)
						}
						walking = false
					}
				}
				var chainErr error
				for i := len(chain) - 1; i >= 0; i-- {
					if t, err := GetType(chain[i], scope, class); err != nil {
						chainErr = err
						break
					} else {
						lastReturnedType = t
					}
				}
				if chainErr != nil && !tolerantGetType {
					return nil, chainErr
				}
			} else if t, err := GetType(stmt, scope, class); err != nil {
				if !tolerantGetType {
					return nil, err
				}
			} else {
				lastReturnedType = t
			}
		}
	}
	return lastReturnedType, nil
}

func (stmts Statements) String() string {
	switch len(stmts) {
	case 0:
		return ""
	case 1:
		return stmts[0].String()
	default:
		return fmt.Sprintf("%s", stdlib.Join[Node](stmts, "\n"))
	}
}

func (stmts Statements) Type() types.Type     { return nil }
func (stmts Statements) SetType(t types.Type) {}
func (stmts Statements) LineNo() int          { return 0 }
func (stmts Statements) File() string         { return "" }

func (stmts Statements) Copy() Node {
	var copy []Node
	for _, stmt := range stmts {
		copy = append(copy, stmt.Copy())
	}
	return Statements(stmts)
}

type Body struct {
	Statements      Statements
	ReturnType      types.Type
	ExplicitReturns []*ReturnNode
	tolerantInfer   bool // true if ReturnType was set via tolerant (error-skipping) mode
	frozen          bool // true if ReturnType is concrete and should not be overwritten
}

// clearCachedTypes resets cached type information on all statements in the body
// so they will be re-inferred on the next analysis pass.
func (b *Body) clearCachedTypes() {
	for _, stmt := range b.Statements {
		clearNodeType(stmt)
	}
}

func clearNodeType(n Node) {
	if n == nil {
		return
	}
	n.SetType(nil)
	switch node := n.(type) {
	case *MethodCall:
		clearNodeType(node.Receiver)
		for _, arg := range node.Args {
			clearNodeType(arg)
		}
		if node.Block != nil && node.Block.Body != nil {
			for _, s := range node.Block.Body.Statements {
				clearNodeType(s)
			}
		}
	case *AssignmentNode:
		for _, l := range node.Left {
			clearNodeType(l)
		}
		for _, r := range node.Right {
			clearNodeType(r)
		}
	case *InfixExpressionNode:
		clearNodeType(node.Left)
		clearNodeType(node.Right)
	case *Condition:
		clearNodeType(node.Condition)
		for _, s := range node.True {
			clearNodeType(s)
		}
		if node.False != nil {
			if stmts, ok := node.False.(Statements); ok {
				for _, s := range stmts {
					clearNodeType(s)
				}
			} else {
				clearNodeType(node.False)
			}
		}
	case *WhileNode:
		clearNodeType(node.Condition)
		for _, s := range node.Body {
			clearNodeType(s)
		}
	case *ReturnNode:
		for _, v := range node.Val {
			clearNodeType(v)
		}
	case Statements:
		for _, s := range node {
			clearNodeType(s)
		}
	case *CaseNode:
		clearNodeType(node.Value)
		for _, w := range node.Whens {
			for _, c := range w.Conditions {
				clearNodeType(c)
			}
			for _, s := range w.Statements {
				clearNodeType(s)
			}
		}
	}
}

// clearUntypedNodes resets nodes that have nil cached types so they'll be
// re-resolved on the next analysis pass. Nodes with valid types are preserved.
func (b *Body) clearUntypedNodes() {
	for _, stmt := range b.Statements {
		clearUntypedNode(stmt)
	}
}

func clearUntypedNode(n Node) {
	if n == nil {
		return
	}
	// Only clear nodes that had no resolved type — these are the ones that
	// need re-resolution. Nodes with types are left intact.
	switch node := n.(type) {
	case *MethodCall:
		if node.Type() == nil {
			// Recurse into args and receiver
			clearUntypedNode(node.Receiver)
			for _, arg := range node.Args {
				clearUntypedNode(arg)
			}
		}
	case *AssignmentNode:
		if node.Type() == nil {
			for _, l := range node.Left {
				clearUntypedNode(l)
			}
			for _, r := range node.Right {
				clearUntypedNode(r)
			}
		}
	case Statements:
		for _, s := range node {
			clearUntypedNode(s)
		}
	}
}

// clearAnyTypeNodes resets nodes that have AnyType so they will be re-resolved
// using the now-refined scope. AnyType is a placeholder from nil declarations
// (e.g., `k = nil`) that gets refined by later assignments. Nodes resolved
// before the refinement have stale AnyType cached.
func clearAnyTypeNode(n Node) bool {
	if n == nil {
		return false
	}
	cleared := false
	if n.Type() == types.AnyType {
		n.SetType(nil)
		cleared = true
	}
	switch node := n.(type) {
	case *MethodCall:
		if clearAnyTypeNode(node.Receiver) {
			cleared = true
		}
		for _, arg := range node.Args {
			if clearAnyTypeNode(arg) {
				cleared = true
			}
		}
		if node.Block != nil && node.Block.Body != nil {
			for _, s := range node.Block.Body.Statements {
				if clearAnyTypeNode(s) {
					cleared = true
				}
			}
		}
	case *AssignmentNode:
		for _, l := range node.Left {
			if clearAnyTypeNode(l) {
				cleared = true
			}
		}
		for _, r := range node.Right {
			if clearAnyTypeNode(r) {
				cleared = true
			}
			// Clear nil-init assignments so they can be re-evaluated after
			// the variable's type has been refined by later assignments.
			if _, isNil := r.(*NilNode); isNil && !node.Reassignment {
				node.SetType(nil)
				cleared = true
			}
		}
	case *InfixExpressionNode:
		if clearAnyTypeNode(node.Left) {
			cleared = true
		}
		if clearAnyTypeNode(node.Right) {
			cleared = true
		}
	case *Condition:
		if clearAnyTypeNode(node.Condition) {
			cleared = true
		}
		for _, s := range node.True {
			if clearAnyTypeNode(s) {
				cleared = true
			}
		}
		if node.False != nil {
			if stmts, ok := node.False.(Statements); ok {
				for _, s := range stmts {
					if clearAnyTypeNode(s) {
						cleared = true
					}
				}
			} else {
				if clearAnyTypeNode(node.False) {
					cleared = true
				}
			}
		}
	case *WhileNode:
		if clearAnyTypeNode(node.Condition) {
			cleared = true
		}
		for _, s := range node.Body {
			if clearAnyTypeNode(s) {
				cleared = true
			}
		}
	case *ReturnNode:
		for _, v := range node.Val {
			if clearAnyTypeNode(v) {
				cleared = true
			}
		}
	case Statements:
		for _, s := range node {
			if clearAnyTypeNode(s) {
				cleared = true
			}
		}
	case *CaseNode:
		if clearAnyTypeNode(node.Value) {
			cleared = true
		}
		for _, w := range node.Whens {
			for _, c := range w.Conditions {
				if clearAnyTypeNode(c) {
					cleared = true
				}
			}
			for _, s := range w.Statements {
				if clearAnyTypeNode(s) {
					cleared = true
				}
			}
		}
	case *BracketAccessNode:
		if clearAnyTypeNode(node.Composite) {
			cleared = true
		}
		for _, arg := range node.Args {
			if clearAnyTypeNode(arg) {
				cleared = true
			}
		}
	}
	return cleared
}

// refineCompositeAnyIdents walks the AST and directly updates IdentNodes
// whose type contains AnyType (e.g., Array(AnyType)) to match the refined
// type from scope. Unlike clearing + re-evaluating, this approach handles
// idents nested inside cached nodes (MethodCall, CaseNode, etc.) that would
// not be re-evaluated on a second GetType pass.
//
// Does NOT recurse into blocks — block bodies have their own type inference
// and idents there refer to the block's scope context.
func refineCompositeAnyIdents(n Node, scope ScopeChain) bool {
	if n == nil {
		return false
	}
	refined := false
	if ident, ok := n.(*IdentNode); ok {
		if types.ContainsAnyType(ident.Type()) && ident.Type() != types.AnyType {
			if local := scope.ResolveVar(ident.Val); local != BadLocal && local.Type() != nil && !types.ContainsAnyType(local.Type()) {
				ident.SetType(local.Type())
				refined = true
			}
		}
		return refined
	}
	switch node := n.(type) {
	case *MethodCall:
		if refineCompositeAnyIdents(node.Receiver, scope) {
			refined = true
			// Receiver type changed — clear MethodCall type so it re-evaluates
			// with the refined receiver on the next GetType pass.
			node.SetType(nil)
		}
		for _, arg := range node.Args {
			if refineCompositeAnyIdents(arg, scope) {
				refined = true
			}
		}
		// Skip blocks — see doc comment above.
	case *AssignmentNode:
		for _, r := range node.Right {
			if refineCompositeAnyIdents(r, scope) {
				refined = true
			}
		}
	case *InfixExpressionNode:
		if refineCompositeAnyIdents(node.Left, scope) {
			refined = true
			node.SetType(nil)
		}
		if refineCompositeAnyIdents(node.Right, scope) {
			refined = true
			node.SetType(nil)
		}
	case *Condition:
		if refineCompositeAnyIdents(node.Condition, scope) {
			refined = true
		}
		for _, s := range node.True {
			if refineCompositeAnyIdents(s, scope) {
				refined = true
			}
		}
		if node.False != nil {
			if stmts, ok := node.False.(Statements); ok {
				for _, s := range stmts {
					if refineCompositeAnyIdents(s, scope) {
						refined = true
					}
				}
			} else {
				if refineCompositeAnyIdents(node.False, scope) {
					refined = true
				}
			}
		}
	case *WhileNode:
		if refineCompositeAnyIdents(node.Condition, scope) {
			refined = true
		}
		for _, s := range node.Body {
			if refineCompositeAnyIdents(s, scope) {
				refined = true
			}
		}
	case *ReturnNode:
		for _, v := range node.Val {
			if refineCompositeAnyIdents(v, scope) {
				refined = true
			}
		}
	case Statements:
		for _, s := range node {
			if refineCompositeAnyIdents(s, scope) {
				refined = true
			}
		}
	case *CaseNode:
		if refineCompositeAnyIdents(node.Value, scope) {
			refined = true
		}
		for _, w := range node.Whens {
			for _, c := range w.Conditions {
				if refineCompositeAnyIdents(c, scope) {
					refined = true
				}
			}
			for _, s := range w.Statements {
				if refineCompositeAnyIdents(s, scope) {
					refined = true
				}
			}
		}
	case *BracketAccessNode:
		if refineCompositeAnyIdents(node.Composite, scope) {
			refined = true
			node.SetType(nil)
		}
		for _, arg := range node.Args {
			if refineCompositeAnyIdents(arg, scope) {
				refined = true
			}
		}
	}
	return refined
}

func (b *Body) InferReturnType(scope ScopeChain, class *Class) error {
	if b.frozen {
		// Even when frozen, propagate variable types to ident nodes that
		// missed their types during tolerant analysis.
		resolveUntypedIdents(b.Statements)
		return nil
	}
	// To guess the right return type of a method, we have to:

	//	1) track all return statements in the method body;

	//  2) chase expressions all the way to the end of the body and wrap that
	//  last expr in a return node if it's not already there, wherein we record
	//  the types of all assignments in a map on the method.

	// Achieving 1) would mean rewalking this branch of the AST right after
	// building it which seems dumb, so instead we register each ReturnNode on
	// the method as the parser encounters them so we can loop through them
	// afterward when m.Locals is fully populated.

	lastReturnedType, err := GetType(b.Statements, scope, class)

	// After the first pass, variables declared with nil (AnyType) have been
	// refined by subsequent assignments. Re-resolve any nodes that still have
	// AnyType so they pick up the refined types from the scope.
	anyCleared := false
	for _, stmt := range b.Statements {
		if clearAnyTypeNode(stmt) {
			anyCleared = true
		}
		// Directly update ident nodes whose type contains AnyType (e.g.,
		// Array(AnyType)) to match the refined scope type. This is done
		// in-place rather than clear+re-evaluate because cached parent
		// nodes prevent re-evaluation of nested idents.
		if refineCompositeAnyIdents(stmt, scope) {
			anyCleared = true
		}
	}
	if anyCleared {
		lastReturnedType, err = GetType(b.Statements, scope, class)
	}
	if err != nil {
		return err
	}
	finalStatementIdx := len(b.Statements) - 1
	finalStatement := b.Statements[finalStatementIdx]
	switch s := finalStatement.(type) {
	case *ReturnNode:
	case *AssignmentNode:
		var ret *ReturnNode
		if s.OpAssignment {
			ret = &ReturnNode{Val: s.Left}
		} else if _, ok := s.Left[0].(*IVarNode); ok {
			ret = &ReturnNode{Val: s.Left}
		} else {
			ret = &ReturnNode{Val: []Node{s.Right[0]}}
		}
		if retType, err := GetType(ret, scope, class); err != nil {
			return err
		} else {
			lastReturnedType = retType
		}
		b.Statements = append(b.Statements, ret)
	default:
		if scope.Name() != Main {
			if finalStatement.Type() != types.NilType {
				ret := &ReturnNode{Val: []Node{finalStatement}}
				if _, err := GetType(ret, scope, class); err != nil {
					return err
				}
				b.Statements[finalStatementIdx] = ret
			} else if len(b.ExplicitReturns) > 0 {
				// Method has explicit returns and ends with nil — still emit
				// return nil so the Optional return type is satisfied.
				ret := &ReturnNode{Val: []Node{finalStatement}}
				ret.SetType(types.NilType)
				b.Statements[finalStatementIdx] = ret
			}
		}
	}
	if len(b.ExplicitReturns) > 0 {
		for _, r := range b.ExplicitReturns {
			t, _ := GetType(r, scope, class)
			if !t.Equals(lastReturnedType) {
				// When one path returns nil and another returns a concrete
				// type, unify them as Optional(T) instead of erroring.
				unified := unifyReturnTypes(lastReturnedType, t)
				if unified != nil {
					lastReturnedType = unified
				} else {
					return NewParseError(r, "Detected conflicting return types %s and %s in method '%s'", lastReturnedType, t, scope.Name())
				}
			}
		}
	}
	// Post-analysis pass: propagate variable types to all ident nodes.
	// During tolerant analysis, block scopes are transient — variables declared
	// in blocks have their types set on scope locals, but ident nodes referencing
	// them may not have been typed (if GetType was skipped due to errors).
	// Walk all assignments to build a type map, then set types on nil-typed idents.
	resolveUntypedIdents(b.Statements)

	b.ReturnType = lastReturnedType
	return nil
}

// resolveUntypedIdents propagates variable types from assignments to all
// ident node references. This ensures ident nodes inside blocks have types
// even when tolerant analysis skipped some GetType calls.
func resolveUntypedIdents(stmts Statements) {
	varTypes := map[string]types.Type{}
	collectVarTypes(stmts, varTypes)
	if len(varTypes) > 0 {
		applyVarTypes(stmts, varTypes)
	}
}

func collectVarTypes(stmts Statements, varTypes map[string]types.Type) {
	for _, stmt := range stmts {
		collectVarTypesNode(stmt, varTypes)
	}
}

func collectVarTypesNode(n Node, varTypes map[string]types.Type) {
	if n == nil {
		return
	}
	switch node := n.(type) {
	case *AssignmentNode:
		// Collect types from LHS ident nodes (if typed)
		for i, l := range node.Left {
			if ident, ok := l.(*IdentNode); ok {
				if ident.Type() != nil && ident.Type() != types.AnyType && ident.Type() != types.NilType {
					varTypes[ident.Val] = ident.Type()
				} else if i < len(node.Right) {
					// Also infer from RHS type: k = replace_next_larger(...)
					// The RHS node's type tells us what k should be.
					rType := node.Right[i].Type()
					if rType == nil {
						// For MethodCalls, check if the Method has a known return type
						if mc, ok := node.Right[i].(*MethodCall); ok && mc.Method != nil && mc.Method.ReturnType() != nil {
							rType = mc.Method.ReturnType()
						}
					}
					if rType != nil && rType != types.AnyType && rType != types.NilType {
						// For nil-init vars reassigned to concrete type, wrap in Optional
						if _, isNil := node.Right[0].(*NilNode); isNil && !node.Reassignment {
							// This is the k = nil declaration — skip, the reassignment
							// will provide the concrete type
						} else if existing, ok := varTypes[ident.Val]; ok {
							// Keep the existing (possibly Optional) type
							_ = existing
						} else {
							varTypes[ident.Val] = rType
						}
					}
				}
			}
		}
		for _, r := range node.Right {
			collectVarTypesNode(r, varTypes)
		}
	case *MethodCall:
		// If the MethodCall's type is nil but its resolved Method has a
		// return type, propagate it. This happens when tolerant analysis
		// skipped SetType on the node but successfully analyzed the method.
		if node.Type() == nil && node.Method != nil && node.Method.ReturnType() != nil {
			node.SetType(node.Method.ReturnType())
		}
		collectVarTypesNode(node.Receiver, varTypes)
		for _, arg := range node.Args {
			collectVarTypesNode(arg, varTypes)
		}
		if node.Block != nil && node.Block.Body != nil {
			collectVarTypes(node.Block.Body.Statements, varTypes)
		}
	case *Condition:
		collectVarTypesNode(node.Condition, varTypes)
		for _, s := range node.True {
			collectVarTypesNode(s, varTypes)
		}
		if node.False != nil {
			if stmts, ok := node.False.(Statements); ok {
				collectVarTypes(stmts, varTypes)
			} else {
				collectVarTypesNode(node.False, varTypes)
			}
		}
	case *WhileNode:
		for _, s := range node.Body {
			collectVarTypesNode(s, varTypes)
		}
	case Statements:
		collectVarTypes(node, varTypes)
	}
}

func applyVarTypes(stmts Statements, varTypes map[string]types.Type) {
	for _, stmt := range stmts {
		applyVarTypesNode(stmt, varTypes)
	}
}

func applyVarTypesNode(n Node, varTypes map[string]types.Type) {
	if n == nil {
		return
	}
	if ident, ok := n.(*IdentNode); ok {
		if ident.Type() == nil || ident.Type() == types.AnyType {
			if t, found := varTypes[ident.Val]; found {
				ident.SetType(t)
			}
		}
		return
	}
	switch node := n.(type) {
	case *MethodCall:
		applyVarTypesNode(node.Receiver, varTypes)
		for _, arg := range node.Args {
			applyVarTypesNode(arg, varTypes)
		}
		if node.Block != nil && node.Block.Body != nil {
			applyVarTypes(node.Block.Body.Statements, varTypes)
		}
	case *AssignmentNode:
		for _, l := range node.Left {
			applyVarTypesNode(l, varTypes)
		}
		for _, r := range node.Right {
			applyVarTypesNode(r, varTypes)
		}
	case *InfixExpressionNode:
		applyVarTypesNode(node.Left, varTypes)
		applyVarTypesNode(node.Right, varTypes)
	case *Condition:
		applyVarTypesNode(node.Condition, varTypes)
		for _, s := range node.True {
			applyVarTypesNode(s, varTypes)
		}
		if node.False != nil {
			if stmts, ok := node.False.(Statements); ok {
				applyVarTypes(stmts, varTypes)
			} else {
				applyVarTypesNode(node.False, varTypes)
			}
		}
	case *WhileNode:
		applyVarTypesNode(node.Condition, varTypes)
		for _, s := range node.Body {
			applyVarTypesNode(s, varTypes)
		}
	case *ReturnNode:
		for _, v := range node.Val {
			applyVarTypesNode(v, varTypes)
		}
	case Statements:
		applyVarTypes(node, varTypes)
	case *BracketAccessNode:
		applyVarTypesNode(node.Composite, varTypes)
		for _, arg := range node.Args {
			applyVarTypesNode(arg, varTypes)
		}
	case *NotExpressionNode:
		applyVarTypesNode(node.Arg, varTypes)
	}
}

// unifyReturnTypes reconciles two differing return types. If one is NilType
// and the other is concrete, returns Optional(concrete). If one is already
// Optional and the other is its inner type or NilType, returns the Optional.
// Returns nil if the types cannot be unified.
func unifyReturnTypes(a, b types.Type) types.Type {
	if a == types.NilType && b != types.NilType {
		if _, ok := b.(types.Optional); ok {
			return b
		}
		return types.NewOptional(b)
	}
	if b == types.NilType && a != types.NilType {
		if _, ok := a.(types.Optional); ok {
			return a
		}
		return types.NewOptional(a)
	}
	if optA, ok := a.(types.Optional); ok {
		if b.Equals(optA.Element) || b == types.NilType {
			return a
		}
	}
	if optB, ok := b.(types.Optional); ok {
		if a.Equals(optB.Element) || a == types.NilType {
			return b
		}
	}
	return nil
}

func (n *Body) String() string {
	return n.Statements.String()
}

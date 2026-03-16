package parser

import (
	"fmt"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type Condition struct {
	Condition  Node
	True       Statements
	False      Node
	elseBranch bool
	TypeGuard  bool // true if this is a `fail/raise unless is_a?` pattern (skip in compilation)
	Pos
	_type      types.Type
}

func (n *Condition) String() string {
	if n.Condition == nil {
		return fmt.Sprintf("(else %s)", n.True)
	}
	if n.False == nil {
		return fmt.Sprintf("(if %s %s)", n.Condition, n.True[0])
	}
	return fmt.Sprintf("(if %s %s %s)", n.Condition, n.True[0], n.False)
}
func (n *Condition) Type() types.Type     { return n._type }
func (n *Condition) SetType(t types.Type) { n._type = t }

func (n *Condition) isBlockGivenGuard() bool {
	if mc, ok := n.Condition.(*MethodCall); ok && mc.MethodName == "block_given?" {
		return true
	}
	return false
}

// isTypeGuard checks if this Condition is a `fail/raise "..." unless type_check`
// pattern. If so, it extracts type information from is_a?/nil? calls and refines
// local variable types. Returns true if the pattern was recognized and handled.
func (n *Condition) isTypeGuard(locals ScopeChain) bool {
	// Must be: fail/raise "..." unless condition
	// AST: Condition{Condition: Not(check), True: [fail/raise(string)]}
	if n.False != nil || len(n.True) != 1 {
		return false
	}
	call, isCall := n.True[0].(*MethodCall)
	if !isCall || call.Receiver != nil {
		return false
	}
	if call.MethodName != "fail" && call.MethodName != "raise" {
		return false
	}
	not, isNot := n.Condition.(*NotExpressionNode)
	if !isNot {
		return false
	}
	// Walk the inner expression to find type guards on AnyType locals.
	guards := extractTypeGuards(not.Arg)
	if len(guards) == 0 {
		return false
	}
	// Apply type refinements.
	for _, g := range guards {
		local, found := locals.Get(g.name)
		if !found || (local.Type() != types.AnyType && local.Type() != nil) {
			continue
		}
		t := g.resolvedType
		if g.nullable {
			t = types.NewOptional(t)
		}
		locals.Set(g.name, &RubyLocal{_type: t})
	}
	return true
}

// typeGuardInfo holds the result of analyzing a single is_a?/nil? guard.
type typeGuardInfo struct {
	name         string     // local variable name
	resolvedType types.Type // type from is_a?(X)
	nullable     bool       // true if nil? was also present (nil? || is_a?)
}

// extractTypeGuards walks a boolean expression tree of is_a?/nil? calls.
// Returns nil if any part of the expression is not a type guard.
func extractTypeGuards(expr Node) []typeGuardInfo {
	switch e := expr.(type) {
	case *MethodCall:
		if e.Receiver == nil {
			return nil
		}
		ident, isIdent := e.Receiver.(*IdentNode)
		if !isIdent {
			return nil
		}
		switch e.MethodName {
		case "is_a?", "kind_of?":
			if len(e.Args) != 1 {
				return nil
			}
			constNode, isConst := e.Args[0].(*ConstantNode)
			if !isConst {
				return nil
			}
			cls, err := types.ClassRegistry.Get(constNode.Val)
			if err != nil {
				return nil
			}
			return []typeGuardInfo{{name: ident.Val, resolvedType: cls.Instance.(types.Type)}}
		case "nil?":
			return []typeGuardInfo{{name: ident.Val, nullable: true}}
		}
		return nil
	case *InfixExpressionNode:
		if e.Operator != "||" && e.Operator != "&&" {
			return nil
		}
		left := extractTypeGuards(e.Left)
		if left == nil {
			return nil
		}
		right := extractTypeGuards(e.Right)
		if right == nil {
			return nil
		}
		// Merge: for `x.nil? || x.is_a?(Integer)`, combine into one entry.
		merged := mergeTypeGuards(append(left, right...))
		return merged
	}
	return nil
}

// mergeTypeGuards combines guards for the same variable. For example,
// {name: "x", nullable: true} + {name: "x", resolvedType: IntType} →
// {name: "x", resolvedType: IntType, nullable: true}.
func mergeTypeGuards(guards []typeGuardInfo) []typeGuardInfo {
	byName := make(map[string]*typeGuardInfo)
	for i := range guards {
		g := &guards[i]
		if existing, ok := byName[g.name]; ok {
			if g.resolvedType != nil {
				existing.resolvedType = g.resolvedType
			}
			if g.nullable {
				existing.nullable = true
			}
		} else {
			byName[g.name] = g
		}
	}
	var result []typeGuardInfo
	for _, g := range byName {
		if g.resolvedType != nil {
			result = append(result, *g)
		}
	}
	return result
}

func (n *Condition) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	// Detect `fail/raise "..." unless x.is_a?(Type)` patterns — runtime
	// type checks that are redundant in Go. Refine local types and treat
	// the entire statement as a no-op.
	if n.isTypeGuard(locals) {
		n.TypeGuard = true
		return types.NilType, nil
	}
	if n.Condition != nil {
		GetType(n.Condition, locals, class)
	}
	t1, err1 := GetType(n.True, locals, class)
	// else clause
	if n.False == nil {
		if n.elseBranch {
			return t1, nil
		}
		return types.NilType, nil
	}
	t2, err2 := GetType(n.False, locals, class)
	if t1 == t2 && err1 == nil && err2 == nil {
		return t1, nil
	}
	// block_given? guards have legitimately different branch types;
	// prefer the false (no-block) branch type as it uses concrete types.
	if n.isBlockGivenGuard() {
		if err2 == nil && t2 != nil {
			return t2, nil
		}
		if err1 == nil && t1 != nil {
			return t1, nil
		}
	}
	// When one branch is AnyType or NilType, use the broader type rather
	// than erroring. This commonly happens in gem code where dynamic
	// variables produce AnyType on one branch.
	if err1 == nil && err2 == nil && t1 != nil && t2 != nil {
		if t1 == types.AnyType || t2 == types.AnyType {
			return types.AnyType, nil
		}
		if t1 == types.NilType {
			return t2, nil
		}
		if t2 == types.NilType {
			return t1, nil
		}
	}
	return nil, NewParseError(n.Condition, "Different branches of conditional returned different types: %s", n)
}

func (n *Condition) Copy() Node {
	copy := &Condition{True: n.True.Copy().(Statements), Pos: Pos{lineNo: n.lineNo}}
	if n.False != nil {
		copy.False = n.False.Copy()
	}
	if n.Condition != nil {
		copy.Condition = n.Condition.Copy()
	}
	return copy
}

type CaseNode struct {
	Value             Node
	Whens             []*WhenNode
	RequiresExpansion bool
	_type             types.Type
	Pos
}

func (n *CaseNode) String() string {
	return fmt.Sprintf("(case %s %s)", n.Value, stdlib.Join[*WhenNode](n.Whens, "; "))
}
func (n *CaseNode) Type() types.Type     { return n._type }
func (n *CaseNode) SetType(t types.Type) { n._type = t }

func (n *CaseNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	// Ensure case value is typed (needed for array-pattern case/when where
	// the array elements must be individually analyzed).
	if n.Value != nil {
		GetType(n.Value, locals, class)
	}
	var (
		t           types.Type
		nilTypeSeen bool
	)

	for _, w := range n.Whens {
		for _, cond := range w.Conditions {
			ct, err := GetType(cond, locals, class)
			if err != nil {
				return nil, err
			}
			if ct.HasMethod("===") {
				n.RequiresExpansion = true
			}
		}
		tw, err := GetType(w, locals, class)
		if err != nil {
			return nil, err
		}

		if tw != nil {
			if tw != types.NilType {
				if t != nil && t != tw {
					// Branches return different types — this case is used
					// as a statement, not an expression. Set type to nil.
					t = nil
					break
				}
				t = tw
			} else {
				nilTypeSeen = true
			}
		}
	}
	if t == nil && nilTypeSeen {
		t = types.NilType
	}
	return t, nil
}

func (n *CaseNode) Copy() Node {
	caseNode := &CaseNode{Value: n.Value.Copy(), RequiresExpansion: n.RequiresExpansion, _type: n._type, Pos: Pos{lineNo: n.lineNo}}
	for _, when := range n.Whens {
		caseNode.Whens = append(caseNode.Whens, when.Copy().(*WhenNode))
	}
	return caseNode
}

type WhenNode struct {
	Conditions ArgsNode
	Statements Statements
	_type      types.Type
	Pos
}

func (n *WhenNode) String() string {
	if n.Conditions == nil {
		return fmt.Sprintf("(else %s)", n.Statements)
	}
	return fmt.Sprintf("(when (%s) %s)", n.Conditions, n.Statements)
}
func (n *WhenNode) Type() types.Type     { return n._type }
func (n *WhenNode) SetType(t types.Type) { n._type = t }

func (n *WhenNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return GetType(n.Statements, locals, class)
}

func (n *WhenNode) Copy() Node {
	return &WhenNode{n.Conditions.Copy().(ArgsNode), n.Statements.Copy().(Statements), n._type, n.Pos}
}

type WhileNode struct {
	Condition Node
	Body      Statements
	Pos
}

func (n *WhileNode) String() string {
	return fmt.Sprintf("(while %s (%s))", n.Condition, n.Body)
}
func (n *WhileNode) Type() types.Type     { return n.Body.Type() }
func (n *WhileNode) SetType(t types.Type) {}

func (n *WhileNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if _, err := GetType(n.Condition, locals, class); err != nil {
		return nil, err
	}
	if _, err := GetType(n.Body, locals, class); err != nil {
		return nil, err
	}
	return types.NilType, nil
}

func (n *WhileNode) Copy() Node {
	return &WhileNode{n.Condition.Copy(), n.Body.Copy().(Statements), n.Pos}
}

type ForInNode struct {
	For    []Node
	In     Node
	Body   Statements
	Pos
}

func (n *ForInNode) String() string {
	return fmt.Sprintf("(for %s in %s (%s))", n.For, n.In, n.Body)
}
func (n *ForInNode) Type() types.Type     { return n.Body.Type() }
func (n *ForInNode) SetType(t types.Type) {}

func (n *ForInNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	var inType types.Type
	inType, err := GetType(n.In, locals, class)
	if err != nil {
		return nil, err
	}
	if !inType.IsComposite() {
		return nil, NewParseError(n, "For loops over %s not supported", inType)
	}
	if t, ok := inType.(types.Hash); ok {
		if len(n.For) != 2 {
			return nil, NewParseError(n, "For loops over hashes must unpack one key and one value")
		}
		for i, v := range n.For {
			ident, ok := v.(*IdentNode)
			if !ok {
				return nil, NewParseError(n, "Not sure how this even successfully parsed")
			}
			if i == 0 {
				locals.Set(ident.Val, &RubyLocal{_type: t.Key})
			} else {
				locals.Set(ident.Val, &RubyLocal{_type: t.Value})
			}
		}
	} else {
		if len(n.For) != 1 {
			return nil, NewParseError(n, "Destructuring subarrays in for loops not supported")
		}
		ident, ok := n.For[0].(*IdentNode)
		if !ok {
			return nil, NewParseError(n, "Not sure how this even successfully parsed")
		}
		locals.Set(ident.Val, &RubyLocal{_type: inType.(types.CompositeType).Inner()})
	}
	for _, v := range n.For {
		GetType(v, locals, class)
	}
	if _, err := GetType(n.Body, locals, class); err != nil {
		return nil, err
	}
	return inType, nil
}

func (n *ForInNode) Copy() Node {
	return &ForInNode{n.For, n.In, n.Body, n.Pos}
}

type BreakNode struct {
	Pos
}

func (n *BreakNode) String() string {
	return fmt.Sprintf("(break)")
}
func (n *BreakNode) Type() types.Type     { return types.NilType }
func (n *BreakNode) SetType(t types.Type) {}

func (n *BreakNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NilType, nil
}

func (n *BreakNode) Copy() Node {
	return n
}

type NextNode struct {
	Val    Node
	Pos
}

func (n *NextNode) String() string {
	if n.Val != nil {
		return fmt.Sprintf("(next %s)", n.Val)
	}
	return fmt.Sprintf("(next)")
}
func (n *NextNode) Type() types.Type     { return types.NilType }
func (n *NextNode) SetType(t types.Type) {}

func (n *NextNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if n.Val != nil {
		if _, err := GetType(n.Val, locals, class); err != nil {
			return nil, err
		}
	}
	return types.NilType, nil
}

func (n *NextNode) Copy() Node {
	c := &NextNode{Pos: Pos{lineNo: n.lineNo}}
	if n.Val != nil {
		c.Val = n.Val.Copy()
	}
	return c
}

type RescueClause struct {
	ExceptionTypes []string
	ExceptionVar   string
	Body           Statements
	Pos
}

type BeginNode struct {
	Body          Statements
	RescueClauses []*RescueClause
	EnsureBody    Statements
	_type         types.Type
	Pos
}

func (n *BeginNode) String() string {
	return fmt.Sprintf("(begin %s)", n.Body)
}
func (n *BeginNode) Type() types.Type     { return n._type }
func (n *BeginNode) SetType(t types.Type) { n._type = t }

func (n *BeginNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	if _, err := GetType(n.Body, locals, class); err != nil {
		return nil, err
	}
	for _, clause := range n.RescueClauses {
		if clause.ExceptionVar != "" {
			locals.Set(clause.ExceptionVar, &RubyLocal{_type: types.RubyErrorType})
		}
		if _, err := GetType(clause.Body, locals, class); err != nil {
			return nil, err
		}
	}
	if n.EnsureBody != nil {
		if _, err := GetType(n.EnsureBody, locals, class); err != nil {
			return nil, err
		}
	}
	return types.NilType, nil
}

func (n *BeginNode) Copy() Node {
	copy := &BeginNode{
		Body:   n.Body.Copy().(Statements),
		_type:  n._type,
		Pos: Pos{lineNo: n.lineNo},
	}
	for _, clause := range n.RescueClauses {
		copy.RescueClauses = append(copy.RescueClauses, &RescueClause{
			ExceptionTypes: clause.ExceptionTypes,
			ExceptionVar:   clause.ExceptionVar,
			Body:           clause.Body.Copy().(Statements),
			Pos: Pos{lineNo: clause.lineNo},
		})
	}
	if n.EnsureBody != nil {
		copy.EnsureBody = n.EnsureBody.Copy().(Statements)
	}
	return copy
}

type LambdaNode struct {
	Block  *Block
	_type  types.Type
	Pos
}

func (n *LambdaNode) String() string       { return fmt.Sprintf("(lambda %s)", n.Block) }
func (n *LambdaNode) Type() types.Type     { return n._type }
func (n *LambdaNode) SetType(t types.Type) { n._type = t }

func (n *LambdaNode) TargetType(locals ScopeChain, class *Class) (types.Type, error) {
	return types.NewProc(), nil
}

func (n *LambdaNode) Copy() Node {
	return &LambdaNode{Block: n.Block.Copy(), _type: n._type, Pos: Pos{lineNo: n.lineNo}}
}

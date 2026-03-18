package parser

import (
	"fmt"
	"strings"

	"github.com/redneckbeard/thanos/stdlib"
	"github.com/redneckbeard/thanos/types"
)

type AssignmentNode struct {
	Left         []Node
	Right        []Node
	Reassignment bool
	OpAssignment bool
	SetterCall   bool
	Pos
	_type        types.Type
}

func (n *AssignmentNode) String() string {
	sides := []interface{}{}
	for _, side := range [][]Node{n.Left, n.Right} {
		var s string
		if len(side) > 1 {
			s = fmt.Sprintf("(%s)", stdlib.Join[Node](side, ", "))
		} else {
			s = side[0].String()
		}
		sides = append(sides, s)
	}
	return fmt.Sprintf("(%s = %s)", sides...)
}
func (n *AssignmentNode) Type() types.Type     { return n._type }
func (n *AssignmentNode) SetType(t types.Type) { n._type = t }

func (n *AssignmentNode) TargetType(scope ScopeChain, class *Class) (types.Type, error) {
	var typelist []types.Type
	for i, left := range n.Left {
		var localName string
		switch lhs := left.(type) {
		case *IdentNode:
			localName = lhs.Val
			GetType(lhs, scope, class)
		case *BracketAssignmentNode:
			// Bracket assignments modify an element, not the variable itself.
			// Resolve the composite type and trigger key refinement if needed.
			if _, err := GetType(lhs, scope, class); err != nil {
				return nil, err
			}
			n.Reassignment = true
			// Type the RHS — in Ruby, arr[i] = val evaluates to val.
			var valType types.Type
			if i < len(n.Right) {
				var err error
				if valType, err = GetType(n.Right[i], scope, class); err != nil {
					return nil, err
				}
				// Refine empty array element type from the assigned value.
				// arr[0] = 5 should refine arr from Array(AnyType) to Array(IntType).
				if valType != nil && valType != types.AnyType {
					if arr, ok := lhs.Type().(types.Array); ok {
						if arr.Element == types.AnyType {
							// Check if the value is a Tuple — promote to SynthStruct
							if tuple, isTuple := valType.(*types.Tuple); isTuple {
								if ident, ok := lhs.Composite.(*IdentNode); ok {
									ss := promoteTupleToSynthStruct(ident.Val, tuple, scope)
									if ss != nil {
										valType = ss
										// Set the type on the RHS ArrayNode so the compiler sees it
										if i < len(n.Right) {
											n.Right[i].SetType(ss)
										}
										scope.RefineVariableType(ident.Val, types.NewArray(ss))
									}
								}
							} else if ident, ok := lhs.Composite.(*IdentNode); ok {
								scope.RefineVariableType(ident.Val, types.NewArray(valType))
							}
						} else if arr.Element != valType {
							return nil, NewParseError(n, "Attempted to assign %s member to %s", valType, arr)
						}
					}
				}
			}
			// Bracket assignment returns the assigned value, not the collection.
			if valType != nil {
				typelist = append(typelist, valType)
			} else {
				typelist = append(typelist, lhs.Type())
			}
			continue
		case *IVarNode:
			GetType(lhs, scope, class)
		case *CVarNode:
			GetType(lhs, scope, class)
		case *GVarNode:
			GetType(lhs, scope, class)
		case *ConstantNode:
			localName = lhs.Val
		case *MethodCall:
			if lhs.Receiver == nil {
				panic("The first pass through parsing should never result in a receiverless LHS method call, but somehow we got here")
			}
		case *SplatNode:
			if ident, ok := lhs.Arg.(*IdentNode); ok {
				localName = ident.Val
			}
		default:
			return nil, NewParseError(lhs, "%s not yet supported in LHS of assignments", lhs)
		}
		var (
			assignedType types.Type
			err          error
		)
		if n.OpAssignment {
			// operator assignments are always 1:1, so nothing to handle here for multiple lhs or rhs
			assignedType, err = GetType(n.Right[i].(*InfixExpressionNode).Right, scope, class)
		} else {
			switch {
			case len(n.Left) > len(n.Right):
				/*

					There are two valid scenarios here: unpacking of an array into
					locals, and assigning from a method that returns a tuple. Note that
					Ruby's behavior in the event of a length mismatch of the two sides is
					to drop the excess values if lhs is shorter than rhs, and to populate
					excess identifiers on lhs with nil of lhs is longer than rhs. There
					is also the perfectly legal option of assigning a single value that
					cannot be deconstructed to multiple variables, which leaves all but
					the first as nil.

				*/
				var rightIndex int
				if i >= len(n.Right) {
					rightIndex = len(n.Right) - 1
				} else {
					rightIndex = i
				}
				t, err := GetType(n.Right[rightIndex], scope, class)
				if err != nil {
					return nil, NewParseError(n, err.Error())
				}
				switch rt := t.(type) {
				case types.Multiple:
					assignedType = rt[i]
				case types.Array:
					if _, ok := n.Left[i].(*SplatNode); ok {
						assignedType = rt
					} else {
						assignedType = rt.Element
					}
				default:
					if i > rightIndex {
						assignedType = types.NilType
					} else {
						assignedType = t
					}
				}
			case len(n.Left) == len(n.Right):
				assignedType, err = GetType(n.Right[i], scope, class)
			case len(n.Left) < len(n.Right):
				// If there's only one lhs element, this is an implicit Array, and
				// needs to get type checked. Otherwise, as discussed above, we throw
				// away any rhs values beyond the length of lhs.
				if len(n.Left) == 1 {
					array := &ArrayNode{Args: ArgsNode(n.Right), Pos: Pos{lineNo: n.Right[0].LineNo()}}
					if at, err := GetType(array, scope, class); err != nil {
						return nil, err
					} else {
						n.Right = []Node{array}
						assignedType = at
					}
				} else {
					assignedType, err = GetType(n.Right[i], scope, class)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		switch lft := left.(type) {
		case *IVarNode:
			lft.SetType(assignedType)
			n.Reassignment = true
			if class != nil {
				ivar := &IVar{_type: assignedType}
				if err = class.AddIVar(lft.NormalizedVal(), ivar); err != nil {
					return nil, NewParseError(n, err.Error())
				}
			}
		case *CVarNode:
			lft.SetType(assignedType)
			n.Reassignment = true
			if class != nil {
				class.AddCVar(lft.NormalizedVal(), &CVar{Name: lft.NormalizedVal(), _type: assignedType})
			}
		case *GVarNode:
			lft.SetType(assignedType)
			n.Reassignment = true
		case *ConstantNode:
			lft.SetType(assignedType)
			// Scoped constant assignment (e.g., Diff::LCS::Foo = expr): resolve
			// the target scope from the namespace.
			if lft.Namespace != "" {
				targetScope := resolveNamespaceScope(lft.Namespace, scope)
				if targetScope != nil {
					constant := &Constant{name: lft.Val, prefix: lft.Namespace + "::"}
					constant._type = assignedType
					constant.Val = n.Right[i]
					targetScope.(ConstantScope).AddConstant(constant)
				}
				// If we can't resolve the scope, silently skip (non-fatal for gems)
			} else if scope.Current().TakesConstants() {
				constant := &Constant{name: lft.Val, prefix: scope.Prefix()}
				constant._type = assignedType
				GetType(left, scope, class)
				constant.Val = n.Right[i]
				scope.Current().(ConstantScope).AddConstant(constant)
			} else {
				scope.Set(localName, &RubyLocal{_type: assignedType})
			}
		case *MethodCall:
			// we should only ever hit this branch for a setter, and thus we have to
			// munge the call to reflect what's actually happening.
			if !strings.HasSuffix(lft.MethodName, "=") {
				lft.MethodName += "="
			}
			lft.Args = []Node{n.Right[i]}
			if _, err := GetType(lft, scope, class); err != nil {
				return nil, err
			}
			n.SetterCall = true
		default:
			// Store lambda blocks directly in scope so .call() resolution works
			if i < len(n.Right) {
				if lambda, ok := n.Right[i].(*LambdaNode); ok {
					lambdaScope := scope.Extend(NewScope("lambda"))
					lambda.Block.Scope = lambdaScope
					method := &Method{
						Name:      localName,
						ParamList: lambda.Block.ParamList,
						Locals:    NewScope(localName),
						Scope:     lambdaScope,
						Body:      lambda.Block.Body,
						Block:     &BlockParam{Name: localName, ParamList: lambda.Block.ParamList},
					}
					lambda.Block.Method = method
					scope.Set(localName, lambda.Block)
					typelist = append(typelist, assignedType)
					continue
				}
			}
			local := scope.ResolveVar(localName)
			if _, ok := local.(*IVar); ok || local == BadLocal || isMethodCallLocal(local) {
				localType := assignedType
				// When a variable's first assignment is nil (e.g. `k = nil`),
				// treat it as a declaration with unknown type rather than
				// locking it to NilType. This allows subsequent assignments
				// (e.g. `k = some_method(...)`) to refine the type.
				if local == BadLocal && assignedType == types.NilType {
					if _, isNil := n.Right[i].(*NilNode); isNil {
						localType = types.AnyType
					}
				}
				newLocal := &RubyLocal{_type: localType}
				// Mark empty arrays and default hashes as refinable for type inference
				if arrayType, ok := assignedType.(types.Array); ok && arrayType.Element == types.AnyType {
					newLocal.MarkAsRefinable()
				}
				if hashType, ok := assignedType.(types.Hash); ok && hashType.HasDefault && hashType.Key == types.AnyType {
					newLocal.MarkAsRefinable()
				}
				if localType == types.AnyType {
					newLocal.MarkAsRefinable()
				}
				scope.Set(localName, newLocal)
			} else {
				if local.Type() == nil {
					loc := local.(*RubyLocal)
					loc.SetType(assignedType)
					if arrayType, ok := assignedType.(types.Array); ok && arrayType.Element == types.AnyType {
						loc.MarkAsRefinable()
					}
					if hashType, ok := assignedType.(types.Hash); ok && hashType.HasDefault && hashType.Key == types.AnyType {
						loc.MarkAsRefinable()
					}
				} else {
					n.Reassignment = true
				}
				if local.Type() != assignedType {
					if rl, ok := local.(*RubyLocal); ok && rl.IsRefinable() && local.Type() == types.AnyType {
						// Variable was declared with nil (AnyType); refine to the concrete type.
						// Keep Optional wrapping — the variable can hold nil, and Go
						// needs a pointer type (*T) to represent that.
						rl.SetType(assignedType)
					} else if arr, ok := local.Type().(types.Array); ok {
						if arr.Element != assignedType {
							return nil, NewParseError(n, "Attempted to assign %s member to %s", assignedType, arr)
						}
					} else if opt, ok := local.Type().(types.Optional); ok && opt.Element.Equals(assignedType) {
						// Allow assigning inner type T to Optional(T) variable (e.g., x ||= default)
						// Variable keeps its Optional type
					} else if _, ok := assignedType.(types.Optional); ok {
						// Allow upgrading to Optional
					} else {
							return nil, NewParseError(n, "tried assigning type %s to local %s in scope %s but had previously assigned type %s", assignedType, localName, scope.Name(), local.Type())
					}
				}
			}
		}
		typelist = append(typelist, assignedType)
	}
	if len(typelist) > 1 {
		return types.Multiple(typelist), nil
	}
	return typelist[0], nil
}

// resolveNamespaceScope walks a "::" separated namespace to find the target
// ConstantScope. Returns nil if the namespace can't be resolved.
func resolveNamespaceScope(namespace string, scope ScopeChain) Scope {
	parts := strings.Split(namespace, "::")
	var current Scope
	for i, part := range parts {
		if i == 0 {
			local := scope.ResolveVar(part)
			if local == BadLocal {
				return nil
			}
			if cs, ok := local.(Scope); ok {
				current = cs
			} else {
				return nil
			}
		} else {
			if cs, ok := current.(ConstantScope); ok {
				inner, err := cs.ConstGet(part)
				if err != nil {
					return nil
				}
				if innerScope, ok := inner.(Scope); ok {
					current = innerScope
				} else {
					return nil
				}
			} else {
				return nil
			}
		}
	}
	if _, ok := current.(ConstantScope); ok {
		return current
	}
	return nil
}

// isMethodCallLocal returns true if the local is a *MethodCall (e.g., an
// ivar accessor returned by Class.Get). Assigning to a name that shadows
// an accessor should create a new local variable, not update the accessor.
func isMethodCallLocal(local Local) bool {
	_, ok := local.(*MethodCall)
	return ok
}

func (n *AssignmentNode) Copy() Node {
	return &AssignmentNode{
		Left:         n.Left,
		Right:        n.Right,
		Reassignment: n.Reassignment,
		OpAssignment: n.OpAssignment,
		Pos:          n.Pos,
		_type:        n._type,
	}
}

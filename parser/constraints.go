package parser

import "github.com/redneckbeard/thanos/types"

// ConstraintKind describes what kind of type evidence was observed for a variable.
type ConstraintKind int

const (
	AssignedType      ConstraintKind = iota // k = value (concrete type)
	AssignedNil                             // k = nil
	NilChecked                              // k.nil?() called on variable
	ElementNilChecked                       // arr[i].nil?() called on element
)

// TypeConstraint records a single piece of type evidence for a variable.
type TypeConstraint struct {
	Kind ConstraintKind
	Type types.Type // for AssignedType: the concrete type
}

// ResolveConstraints examines all collected constraints for a variable and
// returns the resolved type. Returns nil if constraints don't change the type.
func ResolveConstraints(constraints []TypeConstraint, currentType types.Type) types.Type {
	if len(constraints) == 0 {
		return nil
	}

	var (
		hasNil          bool
		hasNilCheck     bool
		hasElemNilCheck bool
		concreteType    types.Type
	)

	for _, c := range constraints {
		switch c.Kind {
		case AssignedNil:
			hasNil = true
		case NilChecked:
			hasNilCheck = true
		case ElementNilChecked:
			hasElemNilCheck = true
		case AssignedType:
			if c.Type != nil && c.Type != types.AnyType {
				concreteType = c.Type
			}
		}
	}

	// AssignedNil + AssignedType(T) → Optional(T)
	if hasNil && concreteType != nil {
		if _, alreadyOpt := concreteType.(types.Optional); !alreadyOpt {
			return types.NewOptional(concreteType)
		}
		return concreteType
	}

	// NilChecked + AssignedType(T) → Optional(T)
	if hasNilCheck && concreteType != nil {
		if _, alreadyOpt := concreteType.(types.Optional); !alreadyOpt {
			return types.NewOptional(concreteType)
		}
		return concreteType
	}

	// ElementNilChecked on Array(T) → Array(Optional(T))
	if hasElemNilCheck {
		if arr, ok := currentType.(types.Array); ok {
			if _, alreadyOpt := arr.Element.(types.Optional); !alreadyOpt {
				return types.NewArray(types.NewOptional(arr.Element))
			}
		}
	}

	// NilChecked alone (no concrete assignment) — variable is used as nillable
	if hasNilCheck && concreteType == nil {
		if currentType != nil && currentType != types.AnyType && currentType != types.NilType {
			if _, alreadyOpt := currentType.(types.Optional); !alreadyOpt {
				return types.NewOptional(currentType)
			}
		}
	}

	return nil
}

// resolveTypeConstraints walks all RubyLocals in scope, resolves their
// constraints, and updates types. Returns true if any type changed.
func resolveTypeConstraints(stmts Statements, scope ScopeChain) bool {
	changed := false
	if ss, ok := scope.Current().(*SimpleScope); ok {
		ss.Each(func(name string, local Local) {
			rl, ok := local.(*RubyLocal)
			if !ok || len(rl.Constraints) == 0 {
				return
			}
			resolved := ResolveConstraints(rl.Constraints, rl.Type())
			if resolved != nil && !resolved.Equals(rl.Type()) {
				rl.SetType(resolved)
				changed = true
			}
		})
	}
	return changed
}

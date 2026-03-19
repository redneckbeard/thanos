package parser

import (
	"fmt"
	"strings"

	"github.com/redneckbeard/thanos/types"
)

// SynthStructs stores synthesized struct types for the compiler to emit.
// Populated during type inference when Tuple types are assigned to array elements.
var SynthStructs []*types.SynthStruct

// ResetSynthStructs clears the global list (for test isolation).
func ResetSynthStructs() {
	SynthStructs = nil
}

// synthStructName derives a Go struct name from a Ruby variable name.
// "links" → "LinksEntry", "nodes" → "NodesEntry"
func synthStructName(varName string) string {
	name := strings.Title(varName) + "Entry"
	return name
}

// findSynthStruct returns an existing SynthStruct with the given name, or nil.
func findSynthStruct(name string) *types.SynthStruct {
	for _, ss := range SynthStructs {
		if ss.Name == name {
			return ss
		}
	}
	return nil
}

// promoteTupleToSynthStruct converts a Tuple type into a SynthStruct,
// registers it in the global list, and returns it. If a SynthStruct with
// the same name already exists, returns that instead.
// The scope is used to detect self-referencing fields (e.g., linked list patterns
// where one element is nil or comes from accessing the same array).
func promoteTupleToSynthStruct(arrayVarName string, tuple *types.Tuple, scope ScopeChain) *types.SynthStruct {
	name := synthStructName(arrayVarName)

	// Return existing if already created
	if existing := findSynthStruct(name); existing != nil {
		return existing
	}

	fields := make([]types.SynthField, len(tuple.Elements))
	ss := types.NewSynthStruct(name, fields)

	for i, elemType := range tuple.Elements {
		fieldName := fmt.Sprintf("Field%d", i)
		fieldType := elemType

		// If the element type is NilType or AnyType, check if this could be
		// a self-reference (e.g., linked list where one field points back to
		// the same array's element type).
		if fieldType == types.NilType || fieldType == types.AnyType {
			// Heuristic: if the array variable holds Array(AnyType) and
			// one of the tuple elements is nil, treat it as a self-reference.
			local := scope.ResolveVar(arrayVarName)
			if local != nil && local != BadLocal {
				if arr, ok := local.Type().(types.Array); ok {
					if arr.Element == types.AnyType || arr.Element == nil {
						fieldType = ss // self-reference
					}
				}
			}
		}

		fields[i] = types.SynthField{Name: fieldName, Type: fieldType}
	}
	ss.Fields = fields

	// Track which module this SynthStruct belongs to so the compiler
	// emits it in the correct package.
	for i := len(scope) - 1; i >= 0; i-- {
		if mod, ok := scope[i].(*Module); ok {
			ss.ModuleName = mod.QualifiedName()
			break
		}
	}

	SynthStructs = append(SynthStructs, ss)
	return ss
}

package parser

import "github.com/redneckbeard/thanos/types"

// nativeMapMethods lists hash methods that have native Go map equivalents
// implemented in compiler/lower.go's nativeMapTransform. A hash is only
// lowered to a native map if ALL its method calls are in this set.
var nativeMapMethods = map[string]bool{
	"[]":       true,
	"[]=":      true,
	"length":   true,
	"size":     true,
	"empty?":   true,
	"clear":    true,
	"has_key?": true,
	"key?":     true,
	"include?": true,
	"member?":  true,
}

// MarkOrderSafeHashes examines all hash-typed variables in scope and marks
// those that only use methods with native Go map equivalents. Returns a set
// of variable names that are safe to compile as native Go maps.
//
// Conservative rule: if ANY hash in scope uses a method not in nativeMapMethods,
// no hashes are lowered. This avoids type mismatches when hashes interact
// (e.g., h1.merge(h2) requires both to be the same type).
func MarkOrderSafeHashes(scope ScopeChain) map[string]bool {
	var safeNames []string
	anyUnsafe := false

	for _, s := range scope {
		ss, ok := s.(*SimpleScope)
		if !ok {
			continue
		}
		ss.Each(func(name string, local Local) {
			rl, ok := local.(*RubyLocal)
			if !ok {
				return
			}
			if rl.Type() == nil {
				return
			}
			h, isHash := rl.Type().(types.Hash)
			if !isHash {
				return
			}
			// DefaultHash always needs the stdlib wrapper
			if h.HasDefault {
				anyUnsafe = true
				return
			}
			if len(rl.Calls) == 0 {
				// No method calls tracked — hash may be used as an argument
				// to functions (e.g. JSON.generate) that expect OrderedMap
				return
			}
			for _, call := range rl.Calls {
				if !nativeMapMethods[call.MethodName] {
					anyUnsafe = true
					return
				}
			}
			safeNames = append(safeNames, name)
		})
	}

	if anyUnsafe {
		return nil
	}
	safe := map[string]bool{}
	for _, name := range safeNames {
		safe[name] = true
	}
	return safe
}

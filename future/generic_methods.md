# Generic Method Compilation

## Problem

Ruby methods are duck-typed — `def lcs(a, b)` accepts any array regardless of element type. When thanos sees the same method called with `[]int` and `[]string`, it currently errors:

```
method 'lcs' called with Array(StringType) for parameter 'a'
but 'a' was previously seen as Array(IntType)
```

DuckInterface handles this for user-defined classes (synthesizing a Go interface), but not for built-in composite types with different element types. The fix: emit Go generics.

## Detection

In `AnalyzeArguments` (parser/methods.go), when a parameter type conflict involves two composite types of the same kind but different element types:

```
Array(IntType) vs Array(StringType)  → generic
Hash(StringType, IntType) vs Hash(StringType, StringType)  → generic (Phase 3)
IntType vs StringType  → error (not composites)
```

Only applies when element types are all `comparable` in Go (int, string, bool, float — types that support `==`).

## Design

### New type: `GenericParam` (types/generic.go)

```go
type GenericParam struct {
    Name       string   // "T", "K", "V"
    Constraint string   // "comparable", "any"
}

func (g GenericParam) GoType() string { return g.Name }
```

When detection fires, the param type changes from `Array(IntType)` to `Array(GenericParam{Name: "T", Constraint: "comparable"})`.

### Parser changes (parser/methods.go)

In `AnalyzeArguments`, before the terminal error:

1. Check `shouldMakeGeneric(existingType, newType)` — both are Array/Hash with different but comparable elements
2. Replace the param type with `Array(GenericParam{"T", "comparable"})`
3. Record on the Method that it has generic params
4. Update the method's scope local

### Compiler changes (compiler/func.go)

When emitting a method with generic params:

1. Add `TypeParams` field to the `FuncType`:
   ```go
   func Lcs[T comparable](seq1, seq2 []T, block lcsBlock) []T
   ```
2. `GetFuncParams` emits `[]T` instead of `[]int` for generic array params
3. `GetReturnType` emits `[]T` when the return derives from the generic param
4. Call sites need no change — Go infers `T` from arguments

### Body compilation

When `T` appears as an array element:
- `seq1[i]` has type `T` — compiled as-is, Go infers
- `seq1[i] == seq2[j]` — works because `T` is `comparable`
- `hash[element]` — works because `comparable` types are valid map keys
- Return `[]T` — propagates through

### Go constraint: `comparable`

Fits the diff-lcs use case perfectly:
- `==` comparison (lcs algorithm core)
- Map key usage (`position_hash` uses elements as keys)
- Iteration, indexing, append — all work on any type

## Phases

### Phase 1: Single type param, arrays only

- Detect `Array(T1)` vs `Array(T2)` conflicts on same param
- Emit `[T comparable]` type param on function
- Propagate `T` through array indexing and return types
- **Unblocks:** diff-lcs showcase test with both int and string arrays

### Phase 2: Return type preservation

Phase 1 may need `[]interface{}` returns in some cases. Phase 2 tracks `T` through the method body so `[]T` is the return type, preserving type safety at call sites.

### Phase 3: Multi-param generics

- Hash params: `[K comparable, V any]`
- Multiple generic params per method
- Nested generics: `[][]T`

## Scope

| What | In scope | Out of scope |
|------|----------|-------------|
| `Array(T)` params | Phase 1 | |
| `comparable` constraint | Phase 1 | |
| `Hash(K, V)` params | | Phase 3 |
| Non-composite generics (`T` alone) | | Future |
| Monomorphization (per-type copies) | | Phase 2 alt |
| Custom constraints | | Not planned |

## Files to modify

| File | Change |
|------|--------|
| `types/generic.go` (new) | `GenericParam` type |
| `parser/methods.go` | Detection in `AnalyzeArguments`, `shouldMakeGeneric()` |
| `compiler/func.go` | `TypeParams` on FuncDecl, generic-aware `GetFuncParams`/`GetReturnType` |
| `compiler/expr.go` | Index expressions on generic arrays |
| `types/array.go` | `MethodReturnType` when element is `GenericParam` |

## Prerequisite

Go 1.18+ (for `ast.FuncType.TypeParams`). Thanos already targets Go 1.23+ in generated go.mod files.

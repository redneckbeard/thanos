# diff-lcs Compilation Blockers

Goal: `require "diff-lcs"` → compile the gem source to Go, so `Diff::LCS.lcs(a, b)` works end-to-end.

## Status: Core blockers resolved (2026-03-18)

Blockers 1-3 from the original list are all resolved. The `Internals.lcs` method now compiles.
One remaining Go type error in the output prevents the compiled code from building:

```
internals.go: cannot use cond (variable of type interface{}) as *LinksEntry value in struct literal
```

## Remaining issue: ternary type narrowing for SynthStruct fields

**Ruby source** (internals.rb:89):
```ruby
links[k] = [k.positive? ? links[k - 1] : nil, i, j]
```

The ternary `k.positive? ? links[k - 1] : nil` compiles to:
```go
var cond interface{}
if k > 0 { cond = links[k-1] } else { cond = nil }
links[k] = &LinksEntry{Field0: cond, ...}
```

`cond` is typed `interface{}` because the Condition node sees two branches: `*LinksEntry` and `nil`.
But `Field0` is `*LinksEntry` (a pointer type, already nilable). The compiled Go needs `cond` to be
`*LinksEntry` so the struct literal typechecks.

**Fix direction**: When a Condition expression has one branch producing a pointer type and the
other producing nil, the result type should be the pointer type (not `interface{}`). Pointer
types in Go are inherently nilable, so `*LinksEntry` already covers both branches.

## Resolved blockers

### Blocker 1: `replace_next_larger` nil-default param ✓
`last_index = nil` now correctly compiles as `last_index *int`. Call sites wrap with `stdlib.Ptr[int](k)`.

### Blocker 2: `position_hash` — Hash.new block type propagation ✓
`Position_hash` returns `*stdlib.DefaultHash[int, []int]` — block types are correctly inferred.

### Blocker 3: `Internals.lcs` — heterogeneous array (Tuple) ✓
Tuples assigned to array elements are promoted to synthesized Go structs (`LinksEntry`).
Self-referencing fields (linked-list pattern) are detected and typed correctly.

## Compilation panics (not needed for `lcs`)

| What | Error |
|------|-------|
| `DefaultCallbacks` class | invalid Go AST (empty class body) |
| `DiffCallbacks` class | nil pointer dereference |
| `Change` class | `self.class()` not supported |
| `ContextChange` class | nil type asserted as Array |
| `LCS.callbacks_for` | `callbacks.new()` not resolved |
| `Internals.analyze_patchset` | complex case/when on Change types |
| `Internals.intuit_diff_direction` | same pattern |

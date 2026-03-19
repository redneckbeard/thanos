# Remaining diff-lcs Blockers

All 4 failing gauntlet tests are diff-lcs cases that compile the same `internals.go`.
Three distinct issues block Go compilation:

## 1. `dup` not implemented on Array (inverse_vector)

**Priority: High — unblocks inverse_vector compilation**

```ruby
inverse = a.dup  # panic: Method not set on MethodCall (a.dup())
```

`dup` is a shallow copy. For arrays in Go: `slices.Clone(a)` or `append([]T{}, a...)`.
Needs a MethodSpec on ArrayType: returns same type, emits clone.

## 2. SynthStruct definitions not emitted (LinksEntry)

**Priority: High — unblocks lcs compilation**

```
links := []*LinksEntry{}           // undefined: LinksEntry
links[*k] = &LinksEntry{...}      // undefined: LinksEntry
```

The type system correctly synthesizes `LinksEntry` (via Tuple/SynthStruct from commit 4cee944),
and the struct IS emitted — but in `main.go`, not in `internals/internals.go` where it's used.
The SynthStruct definition needs to be emitted in the same package as the code that references it.
Gap is in how the compiler decides which package to place SynthStruct definitions in.

## 3. nil-init refinement doesn't fire in gem tolerant mode (k = nil)

**Priority: Medium — requires freeze/tolerant interaction fix**

```
k := nil    // should be: var k *int
```

The nil-init → `var x T` refinement (commit 391dab6) works for user methods but not inside
gem method bodies. `k` is assigned nil, then later `k = Replace_next_larger(...)` which
returns `*int`. The refinement doesn't fire because Replace_next_larger's return type isn't
resolved during tolerant analysis. May need the frozen return type mechanism to propagate
Replace_next_larger's concrete return type into lcs's body.

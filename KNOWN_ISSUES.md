# Known Issues

## parser/

### Method param type conflict between nil and concrete type

Calling the same method with `nil` and a concrete type for the same parameter causes a parse error rather than inferring `Optional(T)`.

```ruby
def test(x)
  x ||= 10
  puts x
end
test(nil)   # first call: NilType
test(3)     # second call: IntType — conflict
```

**Root cause**: `AnalyzeArguments` treats `NilType` vs `IntType` as a type conflict. It should recognize this as `Optional(IntType)` = `*int`.

## compiler/

### Gem compilation error messages are opaque

When a gem method references an unimplemented method (e.g., `a.dup()`), the error surfaces as `Method not set on MethodCall (a.dup())` rather than a clear message like "unresolved method 'dup' on Array(IntType)". The original analysis error is swallowed by tolerant mode; the compiler hits a nil Method field and panics. Visible in diff-lcs output for `DiffCallbacks`, `Internals.intuit_diff_direction`, etc.

## Future improvements

### `stdlib.Ptr[T](expr)` lowering to `&v` locals

`stdlib.Ptr[T](expr)` is emitted when wrapping a concrete value for an `*T` context. A post-compilation lowering pass could replace this with a local variable and address-of operator, eliminating the stdlib dependency for pointer wrapping:

```go
// Before:
stdlib.Ptr[int](len(enum) - 1)

// After:
v := len(enum) - 1
&v
```

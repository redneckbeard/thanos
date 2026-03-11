# Plan: Scoped Nil Support via Pointer Types

## Design Philosophy

Nil-ability is introduced **only where Ruby code actually uses nil**. Most variables stay as plain Go types. When nil appears, the type is "upgraded" to a pointer type (`*string`, `*int`, etc.), and operations like `compact` strip the pointer back to the value type.

This matches what a human would do when porting Ruby to Go.

## Go Representation

```
Ruby nil        => Go nil (pointer)
Ruby "hello"    => Go ptrTo("hello")   -- only in nil-capable context
Ruby [1, nil]   => Go []*int{ptrTo(1), nil}
arr.compact     => Go []int            -- pointers stripped
```

Helper in stdlib:
```go
func PtrTo[T any](v T) *T { return &v }
```

## Implementation Phases

### Phase 1: Optional Type in the Type System

**New file: `types/optional.go`**

```go
type Optional struct {
    Inner    Type
    Instance instance
}

func NewOptional(inner Type) Type {
    return Optional{Inner: inner, Instance: OptionalClass.Instance}
}

func (t Optional) GoType() string      { return "*" + t.Inner.GoType() }
func (t Optional) IsComposite() bool   { return true }
func (t Optional) Inner() Type         { return t.Inner }
func (t Optional) Outer() Type         { return Optional{} }
// ... remaining Type interface methods
```

Key behavior: `Optional` wraps any inner type and produces a pointer Go type.

**Files to modify:**
- `types/optional.go` (new) — Optional type definition + method specs
- `types/types.go` — register Optional in type system

### Phase 2: Nil Literal Typing

Currently `NilNode.Type()` returns `NilType` (a simple sentinel). We need nil to participate in type inference.

**Inference rule:** When nil appears alongside typed values, infer `Optional[T]`:
- `[1, nil, 3]` → `[]Optional[int]` = `[]*int`
- `x = "hello"; x = nil` → `Optional[string]` = `*string`

**Files to modify:**
- `parser/composites.go` — ArrayNode.TargetType: when an element is NilType, upgrade all elements to Optional[T] instead of rejecting as heterogeneous
- `parser/assignment.go` — when assigning nil to a previously-typed variable, upgrade its type to Optional[T]

### Phase 3: Nil Literal Compilation

**`nil` in array context:**
```ruby
arr = [1, nil, 3]
```
```go
arr := []*int{stdlib.PtrTo(1), nil, stdlib.PtrTo(3)}
```

Non-nil values in an Optional context need wrapping with `PtrTo()`.

**`nil` standalone:**
```ruby
x = nil
```
```go
var x *string  // type inferred from later usage
```

**Files to modify:**
- `compiler/expr.go` — compile nil literal as Go `nil`; compile non-nil values in Optional context with `PtrTo()` wrapper
- `stdlib/optional.go` (new) — `PtrTo[T]()` helper

### Phase 4: Array#compact

The payoff. `compact` takes `[]*T` and returns `[]T`.

```ruby
arr = ["a", nil, "b"]
cleaned = arr.compact
```
```go
arr := []*string{stdlib.PtrTo("a"), nil, stdlib.PtrTo("b")}
cleaned := stdlib.Compact(arr)
```

**Type transition:** `Array(Optional(String))` → `Array(String)`

**Files to modify:**
- `types/array.go` — add `compact` MethodSpec:
  - ReturnType: unwrap Optional from element type
  - TransformAST: call `stdlib.Compact(rcvr)`
- `stdlib/array.go` (or `stdlib/optional.go`) — generic `Compact[T]()`:
  ```go
  func Compact[T any](arr []*T) []T {
      result := make([]T, 0, len(arr))
      for _, v := range arr {
          if v != nil {
              result = append(result, *v)
          }
      }
      return result
  }
  ```

### Phase 5: Truthiness and Conditionals

Ruby: `if x` is false when x is nil or false.
Go: `if x` requires a bool.

When `x` is `Optional[T]`, `if x` compiles to `if x != nil`.

```ruby
if name
  puts name
end
```
```go
if name != nil {
    fmt.Println(*name)
}
```

Note: inside the truthy branch, `name` is known non-nil, so it could be auto-dereferenced. This is a **type narrowing** feature — within the `if` body, `name` acts as `T` not `*T`. This is the hardest part.

**Files to modify:**
- `compiler/stmt.go` — when condition expr type is Optional, wrap with `!= nil`
- Possibly: scope-level type narrowing for the true branch (stretch goal)

### Phase 6: The `||` Default Pattern

```ruby
name = params[:name] || "Anonymous"
```

When LHS is `Optional[T]` and RHS is `T`:
```go
name := params["name"]  // *string
if name == nil {
    name = stdlib.PtrTo("Anonymous")
}
// or: unwrapped version
```

Better: compile to a helper:
```go
name := stdlib.OrDefault(params["name"], "Anonymous")
```

```go
func OrDefault[T any](val *T, def T) T {
    if val != nil {
        return *val
    }
    return def
}
```

Return type: `T` (unwrapped) — the `||` with a concrete default strips the Optional.

**Files to modify:**
- `types/optional.go` — define `||` method on Optional that returns inner type when RHS is concrete
- `compiler/expr.go` — handle InfixExpressionNode with LOGICALOR where LHS is Optional
- `stdlib/optional.go` — `OrDefault[T]()` helper

### Phase 7: Safe Navigation Operator `&.`

Already parsed! `MethodCall.Op` captures `"&."`. Just needs compiler support.

```ruby
user&.name
```
```go
var name *string
if user != nil {
    name = stdlib.PtrTo(user.Name())
}
```

Return type: `Optional[T]` where T is the method's normal return type.

**Files to modify:**
- `compiler/expr.go` — in CompileExpr for MethodCall, check `n.Op == "&."` and wrap in nil-check
- `compiler/stmt.go` — same for statement context

### Phase 8: Hash#[] with nil awareness (stretch)

Currently `hash[key]` returns the value type. In Ruby, missing keys return nil.

This is a stretch goal because it changes the semantics of ALL hash lookups. Only do this if the hash value type is already Optional, or if the code uses the result in a nil-checking context.

---

## Suggested Implementation Order

1. **Phase 1 + 3 (stdlib):** Optional type + PtrTo helper. Foundation.
2. **Phase 2 (array literals):** `[1, nil, 3]` produces `[]*int`. First visible result.
3. **Phase 4 (compact):** The motivating use case. Test: `[1,nil,2].compact` → `[1,2]`.
4. **Phase 5 (truthiness):** `if x` where x is Optional.
5. **Phase 6 (|| default):** Common Ruby pattern.
6. **Phase 7 (&.):** Already parsed, "just" needs compilation.
7. **Phase 8 (Hash):** Stretch.

## Risks

- **Pointer dereferencing noise:** `*name` everywhere is ugly. Type narrowing (Phase 5) helps but is complex.
- **PtrTo() wrapping:** Every non-nil value in an Optional context needs wrapping. Could be noisy.
- **Infection spread:** If a function returns `*string`, callers need to handle it. How far does Optional propagate?
- **Interaction with existing type inference:** The RefineVariable mechanism works on type identity. `Optional[String]` != `String`. Need to ensure refinement works across the Optional boundary.

## Test Plan

Gauntlet tests for each phase:
```ruby
# Phase 2+3: nil in arrays
gauntlet("nil in array literal") do
  arr = [1, nil, 3]
  arr.each { |x| puts x.nil? ? "nil" : x.to_s }
end

# Phase 4: compact
gauntlet("Array#compact") do
  arr = ["a", nil, "b", nil, "c"]
  puts arr.compact.join(", ")
end

# Phase 5: truthiness
gauntlet("nil truthiness") do
  x = nil
  x = "hello"
  if x
    puts x
  else
    puts "was nil"
  end
end

# Phase 6: || default
gauntlet("nil || default") do
  x = nil
  x = "hello"  # type inference
  y = x || "default"
  puts y
end
```

# Slice Return Propagation

## Problem

Ruby arrays are mutable references. When a method receives an array and mutates it (via `<<`, `push`, `delete`, `compact!`, etc.), the caller sees the changes. Go slices pass the header by value — modifications to existing elements are visible, but `append` (which can reallocate the backing array) is not.

```ruby
def push_stuff(arr)
  arr << 5
  arr
end

a = [1, 2, 3]
push_stuff(a)
puts a.length  # => 4
```

```go
// Naive translation (broken):
func PushStuff(arr []int) []int {
    arr = append(arr, 5)
    return arr
}

a := []int{1, 2, 3}
PushStuff(a)
fmt.Println(len(a)) // => 3 (wrong!)
```

## Strategy: Return-and-Reassign

When a method receives a slice parameter AND mutates it in a way that can change the slice header (append, delete, etc.), include that parameter in the return tuple. At the call site, reassign.

```go
// Fixed translation:
func PushStuff(arr []int) ([]int, []int) {
    arr = append(arr, 5)
    return arr, arr  // original return + mutated param
}

a := []int{1, 2, 3}
_, a = PushStuff(a)
fmt.Println(len(a)) // => 4 (correct)
```

## Detection

A slice parameter needs return propagation when the method body contains any of:
- `<<` / `push` / `append` / `unshift` / `prepend` on the parameter
- `delete` / `delete_at` / `reject!` / `select!` / `compact!` / `uniq!` on the parameter
- `concat` on the parameter
- Any operation that compiles to `append()` or reslicing (`arr = arr[:n]`)

Operations that do NOT require propagation (they modify existing elements, not the header):
- `arr[i] = value` (bracket assignment to existing index)
- `sort!` (in-place sort of existing elements)
- `reverse!` (in-place reversal)
- `map!` (in-place element replacement)
- `each` / `each_with_index` (read-only iteration)

## Scope

Only slice types need this treatment. Other reference-like types are fine:
- **Maps**: Already reference types in Go (pointer to internal hash table). No action needed.
- **Pointers/structs**: Already reference semantics. No action needed.
- **Strings**: Immutable in both Ruby and Go. No action needed.

## Complications

### Transitive mutation
If `foo(arr)` calls `bar(arr)` which calls `baz(arr)`, and `baz` mutates, then all three need the return-reassign treatment. Detection must be transitive — analyze the full call graph for slice params.

### Multiple mutated params
`def merge(a, b)` where both are mutated → return `(original_return, a, b)`. The return tuple grows but remains correct.

### Methods that already return the mutated array
Common Ruby pattern: `def build(arr); arr << 1; arr; end`. Here the return value IS the mutated array. Detect this case and avoid adding a redundant return — just reassign from the existing return value at the call site.

### Block mutations
Blocks that capture and mutate an outer-scope array: `items.each { |x| results << x }`. If `results` is a method parameter, this counts as mutation. The block doesn't change the detection — we're looking at what happens to the parameter within the method body (including blocks).

### Interface compliance
If a translated method implements a Go interface, changing its return signature would break compliance. For thanos-generated code this is manageable since we control both sides, but worth noting.

## Implementation Sketch

### Analysis phase (parser)
1. For each method, identify params with Array type.
2. Walk the method body for mutation operations on those params (including through blocks).
3. Mark the method as having "mutated slice params" with which params are affected.
4. Transitively propagate: if method A passes its slice param to method B which mutates it, A also mutates.

### Compilation phase (compiler)
1. For methods with mutated slice params, append those params to the return type tuple.
2. Add return statements that include the mutated params.
3. At call sites, destructure the return to reassign the mutated args.

### Example transformations

Single mutated param, no original return:
```ruby
def add_item(list, item)    →    func AddItem(list []string, item string) []string {
  list << item                        list = append(list, item)
end                                   return list
                                  }
                                  // call: list = AddItem(list, item)
```

Single mutated param with return value:
```ruby
def add_and_count(list, x)  →    func AddAndCount(list []int, x int) (int, []int) {
  list << x                           list = append(list, x)
  list.length                         return len(list), list
end                               }
                                  // call: n, list = AddAndCount(list, x)
```

No mutation (no change needed):
```ruby
def sum(arr)                →    func Sum(arr []int) int {
  arr.sum                            total := 0
end                                  for _, v := range arr { total += v }
                                     return total
                                 }
```

# Thanos

Thanos is a source-to-source compiler that translates Ruby into human-readable Go. It's designed as a porting aid — the output is meant to be a starting point for a human refactor, not a drop-in runtime replacement.

I started this project in 2021 with lots of ideas, and the slow, tedious pace of progress, coupled with the constraints of having a real job, led me to abandon it. Enter Opus 4.6 -- with very little steering from me, in the course of 3 weeks, most of my ideas are now fully functional. Robots are neat.

## Demo

`examples/showcase.rb` fetches two CSV files from GitHub over HTTPS, diffs them using the diff-lcs gem, and outputs a JSON summary:

```ruby
require 'net/http'
require 'csv'
require 'diff-lcs'
require 'json'

def fetch_csv(host, path)
  body = ""
  Net::HTTP.start(host, 443, use_ssl: true) do |http|
    response = http.get(path)
    body = response.body
  end
  body
end

def csv_to_lines(text)
  table = CSV.parse(text, headers: true)
  lines = []
  table.each do |row|
    lines << row.fields.join(",")
  end
  lines
end

base = "/redneckbeard/thanos/main/examples"
host = "raw.githubusercontent.com"

puts "Fetching CSVs from GitHub..."
v1_text = fetch_csv(host, base + "/students_v1.csv")
v2_text = fetch_csv(host, base + "/students_v2.csv")

puts "Parsing CSV data..."
v1_lines = csv_to_lines(v1_text)
v2_lines = csv_to_lines(v2_text)

puts "v1: " + v1_lines.length.to_s + " rows"
puts "v2: " + v2_lines.length.to_s + " rows"

puts ""
puts "Running diff..."
common = Diff::LCS.lcs(v1_lines, v2_lines)
matching = common.length
total = v1_lines.length
mismatched = total - matching

diffs = {}
i = 0
while i < total
  if v1_lines[i] != v2_lines[i]
    diffs[(i + 1).to_s] = v1_lines[i] + " -> " + v2_lines[i]
  end
  i += 1
end

report = {
  total_lines: total.to_s,
  matching_lines: matching.to_s,
  mismatched_lines: mismatched.to_s,
  diffs_by_line: diffs.to_json
}

puts ""
puts "=== Diff Report ==="
puts report.to_json
```

Compile and run it:

```sh
thanos exec -v 0 -f examples/showcase.rb
```

Output:

```
--------------------
Fetching CSVs from GitHub...
Parsing CSV data...
v1: 10 rows
v2: 10 rows

Running diff...

=== Diff Report ===
{"total_lines":"10","matching_lines":"7","mismatched_lines":"3","diffs_by_line":"{\"2\":\"2,Bob,87,B+ -> 2,Bob,90,A-\",\"4\":\"4,Dave,78,C+ -> 4,Dave,81,B-\",\"8\":\"8,Heidi,74,C -> 8,Heidi,79,C+\"}"}
```

That output was produced by compiled Go, not Ruby. The generated Go for the showcase is at [examples/showcase_go.md](examples/showcase_go.md).

## Commands

```
thanos compile -s <file.rb>          # compile Ruby to Go, print to stdout
thanos compile -s <file.rb> -t dir/  # compile to directory (for multi-file output)
thanos exec -f <file.rb>             # compile and immediately run
thanos test                          # run gauntlet tests (593 passing)
thanos test -f <file.rb>             # run tests from a single file
thanos report                        # show missing methods on built-in types
```

Global flags: `-v 0` suppresses warnings, `--no-gems` disables gem resolution.

## Testing

In addition to more conventional tests for lexer, parser, and compiler components, thanos has two frameworks for ensuring it meets expectations for target Go style and functionality:

**Style tests** (`go test ./compiler`): Ruby input in `compiler/testdata/ruby/` is compiled and compared against expected Go output in `compiler/testdata/go/`. Validates formatting and structure.

**Gauntlet tests** (`thanos test`): End-to-end verification that Ruby stdout matches Go stdout. Written with the `gauntlet` pseudo-method in `tests/*.rb`.

## Limitations

- No metaprogramming (`method_missing`, `define_method`, `send`, `eval`)
- Heterogeneous arrays are only supported in [specific contexts](#how-are-heterogeneous-arrays-handled); heterogenous hashes are not at all
- Type inference requires tracking calls to literal values; library code called only externally may need help
- No Fiber, Thread, or concurrency primitives

## How it works

### Grammar

The yacc grammar ([`parser/ruby.y`](parser/ruby.y)) covers roughly 85% of CRuby's non-metaprogramming grammar rules. Supported: all control flow (`if`/`unless`/`while`/`until`/`for`/`case`/`when`/`case`/`in`), class/module/def with inheritance and mixins, blocks (`{}` and `do`/`end`), exception handling (`begin`/`rescue`/`ensure`/`raise`/`retry`), splat and double-splat parameters, destructured block parameters, regex literals with flags, heredocs, string interpolation, lambdas (all three forms), ranges, safe navigation (`&.`), `||=`, endless methods (`def foo = expr`), and `%w[]`/`%i[]` word arrays.

Notable exclusions: inline rescue (`x = foo rescue default`), `redo`, dynamic symbols (`` :"#{expr}" ``), `%W[]`/`%I[]` interpolated word arrays, block-local variable declarations (`|x; local|`), and `::Foo` top-level constant references. These are commented out in the grammar with their CRuby rule for reference.

### Type inference

Whole-program type inference is performed by [`Root.Analyze()`](parser/root.go#L1296). It tracks method calls back to literal values and constructor calls, propagating types through assignments, returns, and block yields.

Variables use constraint-based inference ([`ResolveConstraints`](parser/constraints.go#L23)). Each local accumulates evidence — `AssignedType`, `AssignedNil`, `NilChecked`, `ElementNilChecked` — and constraints are combined to determine the final type. For example, a variable assigned both `nil` and an `int` resolves to `Optional(int)`, which compiles to `*int` in Go.

A post-analysis pass ([`propagateTypeWidenings`](parser/widening.go#L23)) handles cross-method type propagation. When consumer code calls `.nil?` on elements of an array returned by another method, the producer's return type is widened from `[]T` to `[]*T` to reflect the nillability that the consumer's usage implies.

### Compilation

[`compiler.Compile`](compiler/compiler.go#L120) translates the type-annotated Ruby AST into `go/ast` nodes. Each Ruby method on a built-in type is defined as a [`MethodSpec`](types/proto.go) with a `TransformAST` function that returns Go statements to prepend and an expression to substitute. Blocks on collection methods unfold to inline `for`-`range` loops rather than closures, producing idiomatic Go. The resulting `go/ast.File` is formatted with `goimports` to produce the final source.

### Gem resolution

When `require 'foo'` has no facade match, thanos resolves the gem via Ruby's `$LOAD_PATH` using [`resolveGemRequire`](parser/program.go#L265). Load paths are discovered by running the system Ruby ([`resolveLoadPaths`](parser/program.go#L219)), with explicit support for rbenv and asdf shims. Paths are cached in `.thanos/load_paths.cache` to avoid repeated subprocess calls.

The gem source is parsed into the same AST with non-fatal error handling — parse panics are caught, unsupported constructs are skipped, and the types that can be inferred are made available to user code. This is how diff-lcs works in the demo above: thanos parses the gem source, compiles the `Lcs` method and its dependencies into a separate Go package, and skips the parts of the gem that use unsupported Ruby features.

### Standard library facades

For Ruby standard library modules where the Go stdlib provides equivalent functionality, thanos uses a JSON-driven facade system ([`RegisterFacade`](types/facade.go#L102)) rather than compiling the Ruby source. Facades are embedded at build time from [`facades/*.json`](facades/).

Three tiers of complexity:

- **Tier 1 — Pure JSON.** Ruby method calls map directly to Go function calls with optional argument casting and error handling. Used by Base64, Digest, SecureRandom, JSON, URI, YAML, Zlib, Shellwords. A [`MethodSpec`](types/facade.go#L190) is synthesized from the JSON at startup.
- **Tier 2 — JSON + Go shim.** A thin adapter function in [`shims/`](shims/) bridges semantic gaps between the Ruby and Go APIs. For example, `shims.JSONParse` wraps `encoding/json` to accept a string and return `map[string]string`, matching the signature that Ruby's `JSON.parse` implies. The JSON facade references the shim function by name.
- **Tier 3 — Programmatic `init()`.** For libraries that need kwargs, conditional return types, or multi-statement AST generation that can't be expressed in JSON. CSV and Net::HTTP use this tier, registering full `MethodSpec` implementations in Go `init()` functions.

When the Go return type differs from the thanos type (e.g., `map[string]string` vs `*stdlib.OrderedMap`), [`buildTypeBridge`](types/facade.go#L465) wraps the expression in the appropriate conversion automatically.

### Comment preservation

Since thanos is a porting aid, the goal is to leave the Ruby source behind. Preserving comments in the Go output reduces the manual work needed after translation.

Ruby comments are collected during lexing ([`lexComment`](parser/lexer.go#L821)) and stored by line number on [`Root.Comments`](parser/root.go#L75). During compilation, each Go statement is tagged with the Ruby line number that produced it. After all statements are compiled, [`stampBlockWithComments`](compiler/comments.go#L250) walks the statement list: for each statement, it calls [`emitCommentsBefore`](compiler/comments.go#L69) to flush any Ruby comments with earlier line numbers as Go comment groups, then assigns a monotonically increasing position to the statement.

Positions are mapped through a synthetic `token.FileSet` with 100,000 lines at 10-byte intervals ([`newCommentState`](compiler/comments.go#L22)). This gives `go/printer` the position ordering it needs to place comments correctly without requiring a real source file. The [`rubyToGoComment`](compiler/comments.go#L101) function handles the `#` → `//` conversion.

### How are heterogeneous arrays handled?

Ruby arrays can hold mixed types. Go slices cannot. Thanos handles this in three specific contexts:

**Tuple promotion to SynthStruct.** When a heterogeneous array literal is assigned to an array element (`arr[i] = [name, score, active]`), [`promoteTupleToSynthStruct`](parser/synthstruct.go#L41) converts the tuple into a synthesized Go struct with typed fields (`Field0 string`, `Field1 int`, `Field2 bool`). The struct includes `Get(i int) interface{}` and `Set(i int, v interface{})` methods for index-based access, and the outer array becomes `[]*NameEntry`. The struct is emitted by [`compileSynthStruct`](compiler/synthstruct.go#L15). This is how diff-lcs's internal linked-list structure compiles — the `[prev, i, j]` triples become `LinksEntry` structs with a self-referential `Field0 *LinksEntry`.

**Pattern matching.** Tuple literals used as subjects in `case`/`in` expressions are destructured element-by-element at compile time. Each element is matched against its corresponding pattern independently.

**String formatting.** The `%` operator with a tuple RHS (`"hello %s, you are %d" % [name, age]`) splats the elements as individual `fmt.Sprintf` arguments.

Outside these contexts, heterogeneous array literals produce a `Tuple` type ([`NewTuple`](types/tuple.go#L20)) that does not support method calls or iteration. Using one where a homogeneous collection is required is a compile-time error.

## Ruby-to-Go impedance mismatches

### How does thanos decide when to use Go generics?

When a method is called with `[]int` at one call site and `[]string` at another, [`AnalyzeArguments`](parser/methods.go#L719) detects the type conflict on the parameter. Before erroring, it calls [`tryMakeGeneric`](parser/methods.go#L911): if both types are arrays of comparable elements (int, string, bool, float), the parameter type is replaced with `Array(GenericParam{T, comparable})`, the return type is generified to match, and the compiler emits `[T comparable]` via [`buildTypeParams`](compiler/func.go#L285). Go infers `T` at each call site.

Example:

```ruby
def count_common(a, b)
  count = 0
  a.each { |x| b.each { |y| count = count + 1 if x == y } }
  count
end

puts count_common([1, 2, 3], [3, 4, 5])
puts count_common(["a", "b"], ["b", "c"])
```

Produces:

```go
func Count_common[T comparable](a, b []T) int {
    // ...
}
```

### How does thanos handle duck typing?

When a method parameter receives two different class types, [`tryBuildDuckInterface`](parser/interface.go#L125) walks the method body via [`findMethodCallsOnParam`](parser/interface.go#L38) to collect every method called on that parameter. It verifies both concrete classes implement all required methods, then synthesizes a Go interface. The parameter type becomes that interface; both classes implicitly satisfy it.

For `respond_to?` guards — methods present on some but not all concrete types — thanos emits type-assertion checks:

```go
if _, ok := callbacks.(interface{ Change(s string) }); ok {
    callbacks.(interface{ Change(s string) }).Change(item)
}
```

Interface method signatures are built by [`BuildInterfaceMethodSignatures`](parser/interface.go#L175) from the first concrete type's analyzed method set.

### How are Ruby's pass-by-reference arrays handled in Go?

Ruby arrays are pass-by-reference; mutations inside a method (`<<`, `push`, `concat`, `delete`, in-place variants) propagate to the caller. Go slices are value-typed headers — `append` inside a function doesn't propagate.

[`detectMutatedSliceParams`](parser/methods.go#L558) walks the method body looking for mutating calls on slice parameters. When found, the Go function signature gains extra return values for each mutated param, and [`augmentReturnsWithSliceParams`](compiler/func.go#L252) appends the parameter identifiers to every `return` statement. At each call site, the LHS is expanded: `x = foo(arr)` becomes `x, arr = foo(arr)`.

### How are Ruby hashes translated?

By default, Ruby hashes compile to `*stdlib.OrderedMap[K, V]` to preserve insertion-order guarantees. But [`MarkOrderSafeHashes`](parser/hash_order.go#L28) runs a lowering pass after analysis: if no hash in a given scope uses order-dependent methods (iteration, `keys`, `values`, `to_a`, `to_json`, etc.), all hashes in that scope compile to native Go `map[K]V` via [`nativeMapTransform`](compiler/lower.go#L25) — with direct bracket access, `len()`, `delete()`, and Go 1.21+ `clear()`.

Hashes with a default value (`Hash.new(0)`) always use `OrderedMap` since the default-value semantics aren't representable in a native map.

### How are Ruby blocks compiled?

Blocks on built-in collection methods (`each`, `map`, `select`, `reject`, `sort_by`, etc.) unfold to inline `for`-`range` loops — not closures. `arr.map { |x| x.upcase }` becomes a loop that appends to a new slice. This is idiomatic Go and avoids the overhead of function call dispatch.

For user-defined methods that `yield`, a function type is synthesized from the inferred block argument and return types. The block compiles to a `func` literal conforming to that type. The method receives the block as a regular function parameter.

### How does nil handling work?

[`ResolveConstraints`](parser/constraints.go#L23) combines evidence from the analysis pass. If a variable is assigned `nil` or checked with `.nil?`, its type becomes `Optional(T)`, which compiles to `*T` in Go. The `||` operator on an `Optional` value uses `stdlib.OrDefault(ptr, fallback)` when the RHS matches the inner type — translating Ruby's `x || default` nil-coalescing idiom. Safe navigation (`&.`) compiles to a nil guard.

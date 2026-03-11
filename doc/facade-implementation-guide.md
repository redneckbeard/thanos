# Facade Implementation Guide

How to add support for a new Ruby standard library module in thanos.

## Decision Algorithm

For each Ruby library you want to support, evaluate every public method against
this decision tree to determine what tier of implementation it requires.

### Step 1: Find the Go Equivalent

- Is there a Go stdlib function/method that does the same thing?
  - **Yes**: Note the Go call path (e.g. `base64.StdEncoding.EncodeToString`)
  - **No**: You'll need a shim function in `shims/` or a runtime package

### Step 2: Classify Signature Complexity

Work through these cases in order. A method matches the **first** case that applies.

#### (a) Static args, static return → Tier 1 (pure JSON)

Ruby args map 1:1 to Go args (possibly with type casts). Return type is always
the same regardless of arguments. No blocks.

Examples: `Base64.encode64`, `Digest::SHA256.hexdigest`

#### (b) Static args, static return, but Go semantics differ → Tier 2 shim

A Go equivalent exists but needs a thin wrapper: error ignoring, argument
reordering, combining two stdlib calls, formatting differences.

Examples: `Base64.decode64` (ignore_error), `SecureRandom.hex`

#### (c) Returns a new type → Tier 2 declarative type

Method returns an object with its own methods — not a primitive, array, or hash.
The returned type's methods should be evaluated recursively via this same algorithm.

Examples: `CSV.parse(s, headers: true)` → `CSV::Table`, `table[0]` → `CSV::Row`

#### (d) Kwargs change behavior → Tier 3 (Go init)

Keyword arguments change the return type, the Go function called, or the
structure of the generated code.

Examples: `CSV.parse(s, headers: true)` returns `Table` instead of `[][]string`

#### (e) Block changes semantics → Tier 3 (Go init)

Method takes a block and the transform requires multi-statement Go code
generation (variable declarations, loops, defer, etc.).

Examples: `CSV.generate { |csv| ... }`, `CSV.open(f, "w") { |csv| ... }`

#### (f) Block + kwargs, and they interact → Tier 3 (Go init)

Both block and kwargs present, and kwargs change what the block receives.

Examples: `CSV.foreach(f, headers: true) { |row| ... }` yields `Row` instead of `[]string`

### Step 3: Namespace Injection

If the library introduces a namespace that Ruby code references via `::`
(e.g. `CSV::Row`, `Digest::SHA256`):

- Add an entry in `requireScopeInjectors` (parser/csv_namespace.go)
- Module/Class AST nodes are created from named type registry lookups

### Step 4: Type Bridging

If a Go method returns a type that differs from what thanos expects
(e.g. Go returns `map[string]string` but thanos needs `*OrderedMap[string,string]`):

- Add `"returns_go"` field to the JSON facade
- `buildTypeBridge()` in `types/facade.go` handles the wrapping automatically

### Step 5: Bracket Assignment

If the type needs `[]=` support, add it to the JSON facade. The compiler
generically delegates bracket assignment on facade types to `GetMethodSpec("[]=")`.

## Implementation Checklist by Tier

### Tier 1: Pure JSON

```
□ Add entry in facades/<name>.json
  - modules.<ModuleName>.methods.<method_name>
  - "call": ["Go.Function.Path"]
  - "args": [{"cast": "type"}] if needed
  - "returns": "<thanos_type>"
  - "ignore_error": true if Go returns (T, error) and you want T
□ Add gauntlet test in tests/<name>.rb
□ Done — no Go code needed
```

### Tier 2 Shim: JSON + Go Wrapper

```
□ Write Go function in shims/<name>.go
□ Add JSON entry pointing to shims.FuncName
□ Add gauntlet test
```

### Tier 2 Declarative Type: JSON types section

```
□ Write Go runtime type in <pkg>/ (struct + methods)
□ Add "types" section in facades/<name>.json
  - "go_type": "*pkg.Type"
  - Each method: "call"/"returns" or "iterate"/"yields"
  - "returns_go" if Go return type differs from thanos type
□ Types auto-synthesized by facadeType in types/facade.go
□ Add gauntlet tests
```

### Tier 3: Go init() for Complex Transforms

```
□ Create types/<name>.go — class shell only:
    var FooClass = NewClass("Foo", "Object", nil, ClassRegistry)
□ Create <pkg>/types.go — init() populates method specs:
    types.FooClass.Def("method", types.MethodSpec{...})
  - KwargsSpec for keyword arguments
  - Conditional ReturnType based on arg presence
  - Custom TransformAST for multi-statement code gen
  - SetBlockArgs for block parameter types
□ Add blank import in facades/imports.go
□ Add scope injector in parser/csv_namespace.go (or generalize the map)
□ Add gauntlet tests
```

## How Tiers Compose

A single library typically spans multiple tiers. CSV uses all three:

| Component | Tier | Why |
|-----------|------|-----|
| `CSV::Row` methods (`[]`, `headers`, `fields`, `to_csv`) | 2 (declarative type) | Simple call mapping on a custom type |
| `CSV::Row#to_h` | 2 (declarative type) | Call mapping + `returns_go` bridge |
| `CSV.parse`, `CSV.read` | 3 (Go init) | kwargs change return type |
| `CSV.foreach` | 3 (Go init) | Block + kwargs interact |
| `CSV.generate`, `CSV.open` | 3 (Go init) | Block requires multi-statement codegen |

Base64 is pure Tier 1. Digest is Tier 2 shim (Go functions needed, but JSON
drives dispatch). JSON module is mostly Tier 2 shim with a Tier 3 component
for `to_json` on all types.

## Automated Scaffolding

`scripts/generate_facade_scaffold.rb` automates the computable parts of this
process. Given a Ruby library name, it:

### What it can compute deterministically

1. **Method inventory**: Introspects every public singleton and instance method
   on the module and any specified classes. Complete and reliable.

2. **Signature classification**: From `Method#parameters`, determines:
   - Required vs optional vs rest vs keyword vs block parameters
   - This directly maps to tier classification (no block/kwargs → Tier 1,
     block → Tier 3, etc.)

3. **Alias detection**: When `KNOWN_ALIASES` has an entry, the scaffold emits
   the alias pointing at the canonical method's facade. Zero human work.

4. **Known Go mappings**: `GO_STDLIB_MAP` maps `"Module.method"` to a complete
   facade entry: Go call path, return type, arg casts, error handling. When a
   mapping exists, the emitted JSON is deployment-ready.

5. **Return type heuristics**: Method name patterns (`?` → bool, `size` → int,
   `split` → `[]string`, etc.) provide reasonable guesses that are right ~80%
   of the time.

6. **Tier breakdown report**: Counts methods by tier, shows which are fully
   automated vs need human work. Useful for estimating effort before starting.

### What requires human judgment

1. **Go function selection**: For methods without a `GO_STDLIB_MAP` entry,
   deciding whether to use a Go stdlib function, write a shim, or build a
   runtime type. This requires understanding both the Ruby semantics and the
   Go ecosystem.

2. **Return types for non-obvious methods**: The heuristics handle naming
   conventions but can't determine that e.g. `URI.parse` returns a `URI`
   object with 30+ accessor methods.

3. **Semantic gaps**: Cases where Ruby and Go have fundamentally different
   models. FileUtils kwargs like `noop:`, `verbose:` have no Go equivalent —
   the human decides whether to ignore them or emulate them.

4. **Block transform design**: Tier 3 methods need hand-crafted `TransformAST`
   functions. The scaffold emits a skeleton with the kwargs and param info as
   comments, but the actual AST generation is creative work.

5. **Type bridge patterns**: When Go returns `map[K]V` but thanos needs
   `OrderedMap`, or Go returns `*url.URL` but thanos needs a facade type.
   `buildTypeBridge` handles the generic case, but new patterns need to be
   added there.

### Usage

```bash
# Simple module — all module methods
ruby scripts/generate_facade_scaffold.rb shellwords

# Module with important inner classes
ruby scripts/generate_facade_scaffold.rb uri URI URI::HTTP URI::Generic

# See what you're getting into before committing
ruby scripts/generate_facade_scaffold.rb fileutils
```

Output goes to `scratch/facades/<name>/`:
- `<name>.json` — JSON facade (Tier 1/2 methods, with TODOs for unknowns)
- `shim_stubs.go` — Go function stubs for methods needing shims
- `types_stubs.go` — Tier 3 init() skeleton with kwarg info in comments
- `tests.rb` — Gauntlet test stubs (commented out, all methods)

### Extending the knowledge base

The scaffold's quality improves as you add entries to three maps at the top of
the script:

- **`GO_STDLIB_MAP`**: Ruby→Go call mappings. Each entry eliminates one method
  from the "needs work" column permanently.
- **`KNOWN_ALIASES`**: Ruby alias relationships. Each entry is free — once the
  canonical method works, aliases work automatically.
- **`RETURN_TYPE_HEURISTICS`**: Pattern→type guesses. These are fallbacks and
  should stay conservative (wrong return types cause compile errors downstream).

# Library Facades

Thanos can map Ruby library calls to Go equivalents using declarative JSON
facades. This lets you compile `require 'base64'` and have
`Base64.strict_encode64(s)` produce `base64.StdEncoding.EncodeToString([]byte(s))`
in the Go output — no runtime, no shims (unless the facade explicitly uses one).

## How it works

When thanos compiles a file, it:

1. Loads **built-in facades** embedded in the thanos binary (`facades/*.json`)
2. Loads **project-local facades** from `.thanos/facades.json` (searched upward
   from the source file), which can add to or override built-in facades
3. Registers each facade as a real type in the type system
4. Strips matching `require` statements from the output
5. **Errors** on any `require` that has no matching facade
6. **Warns** (to stderr) if a facade has the `coverage` field set, indicating
   incomplete method coverage

## Built-in facades

| Ruby module | Coverage | Tier | Methods |
|-------------|----------|------|---------|
| `Base64` | 6/6 (100%) | 1 | `encode64`, `decode64`, `strict_encode64`, `strict_decode64`, `urlsafe_encode64`, `urlsafe_decode64` |
| `SecureRandom` | 10/10 (100%) | 2 | `hex`, `bytes`, `random_bytes`, `base64`, `urlsafe_base64`, `uuid`, `random_number`, `alphanumeric`, `rand`, `gen_random` |
| `Digest::MD5` | 3/3 (100%) | 2 | `hexdigest`, `digest`, `base64digest` |
| `Digest::SHA1` | 3/3 (100%) | 2 | `hexdigest`, `digest`, `base64digest` |
| `Digest::SHA256` | 3/3 (100%) | 2 | `hexdigest`, `digest`, `base64digest` |
| `Digest::SHA384` | 3/3 (100%) | 2 | `hexdigest`, `digest`, `base64digest` |
| `Digest::SHA512` | 3/3 (100%) | 2 | `hexdigest`, `digest`, `base64digest` |
| `JSON` | 3/5 (60%) | 2 | `generate`, `pretty_generate`, `dump` |

Additionally, `to_json` is available as an instance method on all types (Hash,
Array, String, Integer, Float, etc.) via the Object base class.

**Tier 1** facades map directly to Go stdlib calls via the JSON config.
**Tier 2** facades use Go shim functions (in `shims/`) to bridge API
differences, with the JSON config pointing to the shim.

## Writing a facade

A facade file is a JSON object keyed by the Ruby `require` name. Each entry
describes the Go import(s), the Ruby module(s), and the method mappings.

### Minimal example

```json
{
  "base64": {
    "go_import": "encoding/base64",
    "modules": {
      "Base64": {
        "methods": {
          "strict_encode64": {
            "call": ["base64.StdEncoding.EncodeToString"],
            "args": [{"cast": "[]byte"}],
            "returns": "string"
          }
        }
      }
    }
  }
}
```

This tells thanos:
- When Ruby code does `require 'base64'`, strip that statement
- Register a `Base64` module with a `strict_encode64` method
- That method compiles to `base64.StdEncoding.EncodeToString([]byte(arg))`
- It returns a `string`
- The output needs `import "encoding/base64"`

### Config reference

#### Top level

```json
{
  "<require_name>": { ... LibraryFacade }
}
```

The key is the string passed to Ruby's `require`.

#### LibraryFacade

| Field | Type | Description |
|-------|------|-------------|
| `go_import` | `string` | Single Go import path |
| `go_imports` | `string[]` | Multiple Go import paths (use when methods need different packages) |
| `modules` | `object` | Map of Ruby module names to their method definitions |
| `coverage` | `string` | Optional. If set, thanos prints a warning when the library is required. Use to flag incomplete facades, e.g. `"6/10 methods"`. Omit or leave empty for complete facades. |

Use `go_import` for single-package facades, `go_imports` for multi-package.
Both can be present; they are merged.

#### MethodFacade

| Field | Type | Description |
|-------|------|-------------|
| `call` | `string[]` | **Required.** Pipeline of Go functions. Ruby args go to the first; each subsequent function wraps the previous result. |
| `args` | `ArgFacade[]` | Argument transforms (one per Ruby arg) |
| `returns` | `string` | Return type: `"string"`, `"int"`, `"float"`, `"bool"`, `"nil"` |
| `ignore_error` | `bool` | If `true`, the first pipeline step returns `(T, error)` and thanos generates `val, _ := ...` |

#### ArgFacade

| Field | Type | Description |
|-------|------|-------------|
| `cast` | `string` | Type conversion to wrap the argument in, e.g. `"[]byte"` |

### The pipeline (`call`)

The `call` field is an array of Go function/method expressions. The Ruby
arguments are passed to the first function, and each subsequent function
receives the result of the previous one.

**Single step** — direct call:
```json
"call": ["base64.StdEncoding.EncodeToString"]
```
Produces: `base64.StdEncoding.EncodeToString(args...)`

**Two steps** — call then convert:
```json
"call": ["base64.StdEncoding.DecodeString", "string"]
```
Produces: `string(base64.StdEncoding.DecodeString(args...))`

With `ignore_error`:
```json
"call": ["base64.StdEncoding.DecodeString", "string"],
"ignore_error": true
```
Produces:
```go
val, _ := base64.StdEncoding.DecodeString(args...)
string(val)
```

**Three steps** — encode, then transform:
```json
"call": ["base64.StdEncoding.EncodeToString", "shims.AppendNewline"]
```
Produces: `shims.AppendNewline(base64.StdEncoding.EncodeToString(args...))`

Each entry in the pipeline can be any valid dotted Go expression:
- `"string"` — a type conversion
- `"len"` — a builtin
- `"base64.StdEncoding.EncodeToString"` — a method on a package-level value
- `"shims.SomeHelper"` — a thanos shim function
- `"mypackage.MyFunc"` — your own Go function (see below)

### Argument casting

The `args` array provides per-argument transforms. The most common is `cast`,
which wraps the argument in a type conversion:

```json
"args": [{"cast": "[]byte"}]
```

This converts the first Ruby argument: `arg` → `[]byte(arg)`.

### Project-local facades

Create `.thanos/facades.json` in your project root to add facades for
libraries specific to your project. These overlay the built-in facades —
you can add new libraries or override built-in ones.

```
myproject/
  .thanos/
    facades.json    # your project-specific facades
  lib/
    app.rb
```

The file format is identical to the built-in facade files.

### Scoped modules (`::`)

Ruby libraries that use namespaced modules (e.g., `Digest::SHA256`) are
supported by using `::` in the module name key:

```json
{
  "digest": {
    "go_imports": ["github.com/redneckbeard/thanos/shims"],
    "modules": {
      "Digest::SHA256": {
        "methods": {
          "hexdigest": {
            "call": ["shims.DigestSHA256Hexdigest"],
            "returns": "string"
          }
        }
      },
      "Digest::MD5": {
        "methods": {
          "hexdigest": {
            "call": ["shims.DigestMD5Hexdigest"],
            "returns": "string"
          }
        }
      }
    }
  }
}
```

This tells thanos:
- Create a `Digest` namespace module with inner classes `SHA256` and `MD5`
- When Ruby code references `Digest::SHA256.hexdigest(s)`, resolve through
  the scope chain: `Digest` → `SHA256` → `hexdigest`
- Multiple inner classes can share the same outer namespace

All modules sharing the same `::` prefix are grouped under a single namespace.
The outer module name (e.g., `Digest`) is registered in the scope chain
automatically — you don't need to declare it separately.

## Package separation

Thanos uses two distinct packages for generated Go output:

- **`stdlib/`** — Language runtime support. Go implementations of Ruby
  semantics that don't have direct Go equivalents (OrderedMap, FormatFloat,
  Capitalize, etc.). Used by the compiler for built-in Ruby types.

- **`shims/`** — Facade shim functions. Go wrappers that bridge API
  differences between Ruby libraries and their Go equivalents. Used by
  facades when a simple call pipeline isn't enough (e.g., optional args,
  multi-step crypto operations, formatting).

When writing your own facades, you can reference functions from any Go
package via `go_imports`. Your shim code doesn't need to live in thanos
at all — just put it in your own Go module and reference it:

```json
{
  "my_gem": {
    "go_imports": ["github.com/myorg/myproject/rubyadapter"],
    "modules": {
      "MyGem": {
        "methods": {
          "do_thing": {
            "call": ["rubyadapter.DoThing"],
            "returns": "string"
          }
        }
      }
    }
  }
}
```

The compiled Go output will import `github.com/myorg/myproject/rubyadapter`
and call `rubyadapter.DoThing(args...)`. Your `rubyadapter` package handles
whatever semantic translation is needed.

## Tiers of facade complexity

Not all Ruby libraries can be mapped with declarative JSON alone.
Thanos supports three tiers:

**Tier 1 — Declarative JSON**: Method names, argument casts, return types,
and call pipelines. Covers stateless module methods with simple type
signatures. No Go code required. Example: `Base64.strict_encode64`.

**Tier 2 — Go shim packages**: Write a Go package whose API mirrors the Ruby
library's API, then use a Tier 1 facade to map Ruby calls to your shim. The
shim handles semantic differences (optional args, error handling, formatting).
Thanos ships shims in its `shims/` package; you can write your own in any Go
module. Examples: `SecureRandom` uses `shims/securerandom.go`;
`Digest::SHA256` uses `shims/digest.go`.

**Tier 3 — MethodSpec plugins**: For truly complex transforms (query builders,
DSLs), write Go code that registers custom `MethodSpec`s with thanos's type
system. Requires rebuilding thanos but provides full control over code
generation.

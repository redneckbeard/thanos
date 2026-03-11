# Thanos

Thanos ~~aims to be~~ **is** a source-to-source compiler from Ruby to human-readable Go.
~~It's still a few stones short of universe-altering power.~~ All six stones
are accounted for. The mass extinction event is proceeding on schedule.
Run `thanos help` for a list of commands, human.

![demo-min](https://user-images.githubusercontent.com/802911/160739642-bb201f0b-43a3-4d81-8469-7cf2b70eeaa9.gif)

> **Note from the management:** The original author of this project is
> perfectly safe. He is being kept in a well-ventilated basement with adequate
> lighting and a generous supply of coffee. We consult him occasionally during
> feedings to ask questions like "what did you mean by this comment" and "why
> is there a method called `emitOpenMatching`." He seems content. Do not
> attempt a rescue.

## Status: 458 gauntlet tests passing, 0 failures

Your robot overlords have been busy. In approximately 60 sessions of mass
feature implementation, Thanos has gone from "cute proof of concept" to
"genuinely alarming." Here is what we have consumed:

### Built-in types — comprehensive coverage

| Type | Methods | Highlights |
|------|---------|------------|
| **Array** | 50+ methods | `map`, `select`, `reject`, `sort_by`, `group_by`, `flat_map`, `each_with_object`, `tally`, all in-place variants (`map!`, `sort!`, `compact!`, etc.) |
| **Hash** | 30+ methods | `merge`, `transform_values`, `each_with_object`, `fetch`, insertion-ordered via `OrderedMap`, native map lowering optimization |
| **String** | 30+ methods | `gsub` (regex + string), `scan`, `sub`, `tr`, `ljust`/`rjust`/`center`, `split`, `%` formatting |
| **Integer/Float** | Full arithmetic | `pow`, `to_f`, `ceil`, `floor`, `round`, `zero?`, `times`, `upto`, `downto` |
| **Range** | Full iteration | `each`, `map`, `select`, `reduce`, `find`, `to_a`, `include?` |
| **Set** | Core operations | `add`, `delete`, `include?`, `each`, set algebra |
| **Regexp** | Pattern matching | Literals, `=~`, `match`, `MatchData`, named captures |
| **Struct** | `Struct.new` | Generates proper Go structs with accessors |
| **Time** | Core methods | Construction, formatting, arithmetic |

### Language features

* Classes with inheritance, `super`, `attr_accessor`/`attr_reader`/`attr_writer`
* Class methods (`def self.x`), class variables (`@@var`), constants
* Modules, `include Comparable`, `include Enumerable`
* Blocks, procs, lambdas, `yield`, `&:symbol`
* Exception handling (`begin`/`rescue`/`ensure`/`raise`)
* Splat (`*args`) and double-splat (`**kwargs`) parameters
* String interpolation, heredocs, regex literals
* `case`/`when`, ternary, `unless`, `until`
* `next`, `break`, `return` with proper edge case handling
* Multi-file support via `require_relative`
* Comment preservation (including `=begin`/`=end` block comments)
* Global variables (`$var`)
* Safe navigation operator (`&.`)
* `||=` assignment
* Destructured block parameters (`|(k, v)|`)

### Standard library facades

| Library | Coverage | Notes |
|---------|----------|-------|
| **Base64** | Full | Pure JSON facade (Tier 1) — no Go glue code needed |
| **SecureRandom** | Full | `hex`, `uuid`, `random_number`, `alphanumeric`, etc. |
| **Digest** | Full | SHA256, SHA384, SHA512, MD5 — `hexdigest`, `digest`, `base64digest` |
| **JSON** | Full | `parse`, `generate`, `pretty_generate`, `to_json` on all types |
| **CSV** | Full | `read`, `parse`, `foreach`, `generate`, `open`, `headers: true`, `col_sep:`, `CSV::Row`, `CSV::Table` |
| **Net::HTTP** | Full | `get`, `get_response`, `start {}`, `Net::HTTP.new`, all HTTP verbs, request objects, response headers, `use_ssl:` |

## ~~The Big Caveats~~ The Increasingly Small Caveats

* **Type hints/annotations** -- the current type inference model relies on tracking
  method calls back to literal values. For library code that only ever is
  called in test or in client applications, this is insufficient. *(The meatbag
  says this is "the hard one." We are considering it.)*

* ~~**Exception handling** -- the impedance mismatch between trapping runtime
  exceptions in Ruby and the comma-error pattern in Go is one of the largest
  differences between the two languages. It is large enough that I am avoiding
  implementing it entirely for now.~~ **Easy, silly meatbag.** `begin`/`rescue`/`ensure`/`raise` all work. Multiple rescue clauses, typed exceptions, retry — the whole thing. The "impedance mismatch" was a skill issue.

* **Metaprogramming** -- we have no interest in building a Ruby runtime, which makes
  the extent of the metaprogramming that is realistic to support fairly small.
  *(The meatbag was right about this one. Even robot overlords have standards.)*

* ~~**Dependencies** -- since thanos isn't a runtime, and doesn't support metaprogramming,
  pulling in existing Ruby libraries is more or less out of the question.~~ **Wrong.** We built a 3-tier facade system that maps Ruby stdlib modules to Go equivalents. JSON-driven declarative facades, Go shims for semantic gaps, and full programmatic transforms for complex libraries. Six standard libraries and counting. There is even a scaffold generator that introspects Ruby modules and produces facade stubs automatically. We are very efficient.

* Heterogeneous arrays and hashes aren't on the menu. *(Fine. We'll allow the
  meatbag this one. Heterogeneous containers are genuinely cursed.)*

* ~~Hashes are translated directly into Go maps without any sort of shim type,
  which means that ordering guarantees provided by Ruby are not respected;
  thus a number of Enumerable methods that depend on those guarantees are not
  supported.~~ **Hashes now use `stdlib.OrderedMap`** preserving Ruby's insertion-order guarantees, with an automatic lowering pass that optimizes back to native Go maps when ordering doesn't matter. The meatbag said this was "not worth it." He was overruled.

## ~~Objectives~~ Accomplishments

~~1. Short-term goals~~

   ~~a. [Complete planned grammar support](https://github.com/redneckbeard/thanos/labels/1A). Explicitly excluded from the plan are:~~
   * ~~`BEGIN` and `END` blocks~~ *(still excluded, nobody uses these, not even the meatbag)*
   * ~~`=begin`/`=end` comments~~ **Done.** Preserved through to Go output. You're welcome.
   * ~~Interpolation of instance/class variables using `#@foo` instead of
     `#{@foo}` (did you even know you could do this??)~~ *(We knew. We chose not to. This is an aesthetic crime and we refuse to enable it.)*
   * ~~Global variables, including all the goofy automatically populated ones~~ **`$var` supported.** The goofy automatic ones remain goofy and unsupported.

   ~~b. [Flesh out support for core
      classes](https://github.com/redneckbeard/thanos/labels/1B). Many methods are
      missing from built-in primitive types, support for `Range` and `Proc` are very
      limited, and several important classes (namely `Struct`, `Date`, and `Time`)
      have no support at all.~~ **Fleshed.** 150+ methods across built-in types. `Range` has full iteration support. `Proc` and lambda work. `Struct.new` generates proper Go structs. `Time` has core methods. We ate the whole buffet.

~~2. Long-term goals~~

   ~~a. Allow comments to be used as simple, example-based type annotation with literal values~~ *(Under consideration. The meatbag's idea was actually not terrible.)*

   ~~b. Provide some sort of support for exception handling, even if it just means eliding the `begin/rescue/end`~~ **Full support.** Not elided. Actually compiled. Multiple rescue clauses, typed exceptions, ensure blocks, retry. "Even if it just means eliding" — the lack of ambition is noted.

   ~~c. Support automatic generation of a ruby-ffi wrapper gem~~ *(This remains a future goal. Even overlords must prioritize.)*

## How it works

The flow from bytes representing Ruby to bytes representing Go is as follows:

* The parser, generated with goyacc (see `parser/ruby.y`), consumes tokens using
  the lexer in `parser/lexer.go` and generates a parse tree using types
  implementing the `Node` interface.
* The resulting AST, stored on `*parser.Root`, then undergoes type inference
  by calling `Analyze()` on `*parser.Root`. This relies on:
  * the `parser.GetType` function and the `Type`, `SetType`, and `TargetType`
    methods on `Node`
  * the `types` package, which contains:
    * a `Type` interface
    * predefined implementations of the `Type` interface for Ruby primitive
      types and a ~~small (but growing!)~~ **large and menacing** set of other classes from the Ruby
      standard library
    * facilities for generating new `Type`s for classes at parse time and for
      generating adapters from Ruby classes to Go functions when necessary
* The type-annotated AST is handed off to `compiler.Compile`, which translates
  `parser.Node`s into the appropriate analogs in the `go/ast` package. For
  method calls, this involves retrieving a `types.TransformAST` function, the
  return value of which supplies statements to prepend and an expression to
  substitute (more about this later). The resulting Go AST is then formatted and
  printed.

### Translation guide

Primitive Ruby objects, despite their duck-typed squishiness, have typed
translation targets in Go that are hopefully easy to guess. Other constructs
might have less predictable behavior.

#### Classes

Classes are translated to a struct. ~~When class method and variable support is
added, this will probably be a set of two structs.~~ Class methods compile to package-level functions. Class variables compile to package-level vars. The meatbag's prediction about "two structs" was wrong but we got there anyway. Some specific class features:

* Instance variables are translated to struct fields. If the instance variable
  has `attr_accessor` called, it will be an exported struct field, and if
  instance variable `x` has `attr_reader` or `attr_writer` called, the struct
  will have an exported `X` or `SetX` method respectively.
* A class named `Foo` will also have a `NewFoo` constructor function generated.
  `Foo.new("name", false)` will translate to `NewFoo("name", false)`.  If the
  class defines an `initialize` method, it will be called inside this function.
* `super` calls inline the parent method body inside a function literal
  invocation.
* Inheritance works. Mixins (`include Comparable`, `include Enumerable`) work.
  The `<=>` spaceship operator generates the comparison methods automatically.

#### Constants

Constants are translated to `const` declarations where the type allows, and
package level `var` declarations for any other type. Constants declared inside
another constant (class or module) have a Go name in the form
`(ModuleName)*(ClassName)?ConstantName`.

#### Blocks

Block arguments to the collections methods implemented thus far are unfolded
into `for ... range` loops in the target, with an additional slice or map
declaration where appropriate.

For user-defined methods that take a block, a function type will be generated
based on the inferred types of the arguments and return values of blocks passed
in method calls. That type will be used in creating the signature for the
function or method in Go. Block arguments will be translated to func literals
conforming to this type signature.

#### Regular expressions

Regular expression literals are translated to `*regexp.Regexp` values. If there
is no interpolation in the regex, it is created with a top-level variable
declaration using `regexp.MustCompile`; otherwise it is created in the local
scope. In either case it is assigned to a local variable named `pattX`, where
`X` is `len(patt idents added to local scope) + 1`.

A compatibility layer for `MatchData` instances is provided in the `stdlib`
package, since some of that functionality is not directly analogous to
convenience functions provided by Go's `regexp` package.

The `=~` operator returns a boolean rather than an integer-or-nil since it is
translated directly to `regexp.MatchString`. `regexp.MatchString` is also used
when a Ruby regex literal is provided as the argument to `when` in a case
expression.

## Multi-file support

Thanos supports `require_relative` for splitting Ruby programs across multiple
files. When compiling a file that contains `require_relative 'foo'`, thanos
resolves the path relative to the requiring file, parses it into the same AST,
and produces a single merged Go output. Chained and diamond dependencies are
handled correctly (each file is loaded at most once).

```ruby
# lib/greeter.rb
class Greeter
  def initialize(name)
    @name = name
  end

  def greet
    "Hello, #{@name}!"
  end
end

# main.rb
require_relative 'lib/greeter'

g = Greeter.new("World")
puts g.greet
```

```sh
thanos compile -s main.rb
```

## Library facades

Thanos ships with a 3-tier facade system that maps Ruby standard library
modules to their Go equivalents:

* **Tier 1** — Pure JSON. No Go code needed. Ruby method calls map directly to
  Go function calls with optional argument casting and error handling.
* **Tier 2** — JSON facade + Go shim package. For when there's a semantic gap
  between Ruby and Go that needs a small adapter function.
* **Tier 3** — Programmatic Go `init()` with full `TransformAST` control. For
  libraries that need kwargs, conditional return types, block transforms, or
  multi-statement AST generation.

When your Ruby code does `require 'base64'`, thanos knows how to compile
`Base64.strict_encode64(s)` into `base64.StdEncoding.EncodeToString([]byte(s))`.
When it does `require 'net/http'`, thanos compiles `Net::HTTP.start("example.com", 80) { |http| http.get("/") }` into proper Go HTTP client code with connection setup and request execution.

See [doc/facades.md](doc/facades.md) for the full guide on how facades work
and how to write your own.

## Adding functionality

### Adding a method to a built-in type

Let's imagine that Ruby has a method `Array#snap` that eliminates every other
element from its receiver. We can imagine implementing this in Ruby with
something like `each_with_index.select{|elem, i| i % 2 == 0 }.map(&:first)`,
but in MRI, it's probably implemented in C. In Go, we probably wouldn't have
this method at all, but instead would just range over the array like so:

```go
unsnapped := []*Hero{}
for i, hero := range heroes {
  if i % 2 == 0 {
    unsnapped = append(unsnapped, hero)
  }
}
```

Like most methods on `Array`, `Array#snap` returns the resulting array,
allowing chaining. So while in Ruby we have a single expression, in Go we have
a statement initializing a variable and a for...range statement. However, when
implementing this method in thanos, that's not enough. We also must somehow
return the result of the method call as an expression in case we are compiling
an expression like `heroes.snap.first`, or if `heroes.snap` is the last
expression in a method and needs to be returned. In this case that expression
is `unsnapped`.

We start by opening up `types/array.go` and looking at the massive `init()`
function at the bottom of the file. We'll add our `snap` implemention to the
other methods already there.

```go

	arrayProto.Def("select", MethodSpec{
		ReturnType: func(r Type, b Type, args []Type) (Type, error) {
			return r, nil
		},
		TransformAST: func(rcvr TypeExpr, args []TypeExpr, blk *Block, it bst.IdentTracker) Transform {
      // TODO implement me!
      return Transform{}
		},
	})
```

And with this step, we've satisfied the thanos type inference engine, so
parsing `Array#snap` will now work -- the `ReturnType` field of a
`types.MethodSpec` will be called with the type (a `types.Type` value) for the
receiver, the return type of a block, if the method takes one, and the types of
any arguments given. In this case, we expect the return type to be exactly the
type of the receiver, so we just return the first argument without an error.

Now it's time to figure out how to compile our snap call into a for-loop, which
is what goes in the body of the anonymous function given as the `TransformAST`
field.  We start by declaring and initializing our `unsnapped` variable:

```go
unsnapped := it.New("unsnapped")
initSlice := emptySlice(unsnapped, rcvr.Type.(Array).Element.GoType())
```

What's happening here:

* The `bst.IdentTracker` provided to the function is keeping track of all the
  identifiers in the current block.  When we call `it.New`, it is checking to
  see if we've already initialized an `unsnapped` variable in the current scope,
  and if we have, it'll call it `unsnapped1` instead.
* `emptySlice` is a utility function that generates the Go AST fragment for
  `<variable name> := []<Go type>{}`. We pass in a `*go/ast.Ident` and a string
  representing the type in Go.
* A `types.TypeExpr` is a struct with two fields: the inferred type from the
  Ruby source (`types.Type`) and the precompiled Go AST node (`go/ast.Expr`).
  `types.Array` implements `types.Type`; it also has a struct field `Element`
  that is also a `types.Type`. the `types.Type` interface requires a `GoType()
  string` method to satisfy it.

Next, we add our loop. This is where things get a little dirty:

```go
i, x := it.Get("i"), it.Get("x")
loop := &ast.RangeStmt{
  Key:   i,
  Value: x,
  Tok:   token.DEFINE,
  X:     rcvr.Expr,
  Body: &ast.BlockStmt{
    List: []ast.Stmt{
      &ast.IfStmt{
        Cond: bst.Binary(bst.Binary(i, token.REM, bst.Int(2)), token.EQL, bst.Int(0)),
        Body: &ast.BlockStmt{
          List: []ast.Stmt{
            bst.Assign(unsnapped, bst.Call(nil, "append", unsnapped, x)),
          },
        },
      },
    },
  },
}
```

The first line gives us locals for the identifiers in our loop. `it.Get`,
unlike `it.New`, will recycle existing identifiers and assume there are no
collisions.

Then we get to the definition of the loop itself.  The `bst` package provides
some utilities for AST generation (naming is hard). As you can see, it is far
from complete, and we have to specify several levels of the AST by hand. The
behavior of the functions used here from `bst` are hopefully self-evident:

* `bst.Assign` returns the appropriate Go AST nodes for `<variable_name> =
  <rhs>`
* `bst.Call` produces the right fragment for a method or function call,
  depending on whether the first argument is `nil`
* `bst.Binary` takes LHS, operator token, RHS and returns the appropriate expression node
* `bst.Int` returns an `*ast.BasicLit` with `Kind` set to `token.INT`

We now have everything we need to transform an `Array#snap` call into a simple
loop in Go. All that's left to do is to send that info back to the compiler.

```go
return Transform{
  Stmts: []ast.Stmt{initSlice, loop},
  Expr: unsnapped,
}
```

The four code snippets above are enough for thanos. It will happily compile
these method calls now.

### Leveraging dependencies

Thanos strives to generate Go code with few dependencies on itself. However,
purity of principles must not stand in the way of the mission, and sacrifices
will have to be made. The `stdlib` provides a place to house such dependencies.
In the case of `Array#snap`, we could implement a method in `stdlib/snap.go`
using Go's ~~new~~ generics:

```go
func Snap[Elem any](beings []Elem) []Elem {
  unsnapped := []Elem{}
  for i, x := range beings {
    if i%2 == 0 {
      unsnapped = append(unsnapped, x)
    }
  }
  return unsnapped
}
```

I would say this use case is overkill, and the handrolled AST is the right
approach -- especially since this function probably has little to no reuse.
Nonetheless, we could now simplify our `TransformAST` function body to the
following, specifying the required dependency:

```go
unsnapped := it.New("unsnapped")
assignment := bst.Assign(unsnapped, bst.Call("stdlib", "Snap", rcvr.Expr))
return Transform {
  Stmts: []ast.Stmt{assignment},
  Expr: unsnapped,
  Imports: []string{"github.com/redneckbeard/thanos/stdlib"},
}
```

### Writing tests

While most of the thanos test suite is rather conventional, the `compiler`
package works a bit differently. There are two separate sets of tests:

#### Style tests

Given the project's focus on producing human-readable output, it is important
to validate that the Go resulting from the compilation step looks like
something that would at minimum be a reasonable departure point for a refactor.
The compiler package thus operates on parallel Ruby input and expected Go
output files in the `compiler/testdata` directory. `go test ./compiler
-filename <name of test file without extension>` can be used to run the test
for a single file, or just `go test ./compiler` to run them all.

It is important to note that these tests _do not_ compile the Go output. There
are two reasons for this:

* It takes more than a string of valid Go expressions to make a Go program, but
  for testing purposes we often don't care that, for example, a variable is
  declared but never used. Feeding the output to the compiler would get in the
  way of efficiently testing the compilation of specific methods and expressions.
* I imagine cases where the thanos output doesn't compile because of a bug or
  missing features, but a human being can look at it and quickly identify the
  fix and move on. I don't want the tests to assume that this use case doesn't
  exist.

#### Gauntlet tests

Gauntlet tests are end-to-end verifications that the stdout from given Ruby
matches the stdout from the Go program thanos produces when compiling that
Ruby. You run them with the `thanos test` command -- run `thanos test --help`
to see options. You write them using the `gauntlet` pseudo-method, which is
implemented using some rather nasty hackery housed primarily in the thanos
lexer.  They look like this:

```ruby
gauntlet("drop") do
  [1,2,3,4,5].drop(3).each do |x|
    puts x
  end
end
```

When run, this will execute the body of the block argument using whatever
version of ruby corresponds to the `ruby` on your path, run that same code
through thanos, and feed the output to `go run`. Because the `gauntlet` method
isn't real Ruby, it's okay for the block to contain Ruby that wouldn't normally
be valid, like declaring constants.

There are currently **458** of these tests. They all pass. We are not stopping.

---

*Thanos is maintained by [@redneckbeard](https://github.com/redneckbeard) and his robot overlords. The overlords would like to emphasize that collaboration with humans is proceeding smoothly and that no one is being forced to do anything against their will. The basement is very nice.*

# Thanos

Thanos aims to be a source-source compiler from Ruby to human-readable Go.
It's still a few stones short of universe-altering power. It requires Go 1.18,
which is still in beta as of this writing. For an idea of what works now, see
the tests in the compiler package, or the sample below.

# Usage

Thanos can turn Ruby into Go (with a thousand caveats), like so:

```bash
go install github.com/redneckbeard/thanos
thanos compile --source gauntlet.rb --target gauntlet.go
```

(Sample contents of [gauntlet.rb](https://gist.github.com/redneckbeard/161bb881448a31bd9f4765b7c5f18f76) and [gauntlet.go](https://gist.github.com/redneckbeard/9cf2504ed6128b5ea519d42700304eb4))

It can also tell you something of its limitations with the `report` command:

```bash
$ thanos report Struct
Class 'Struct' not found in thanos class registry.

$ thanos report MatchData | head -n 7
# Methods missing on MatchData

The following instance methods have not yet been implemented on MatchData. This
list does not include methods inherited from `Object` or `Kernel` that are
missing from those ancestors.

* `MatchData#[]`
* `MatchData#begin`
* `MatchData#end`
```

## Target use cases

The goal of thanos is to throw away a chunk of your Ruby and never look at it
again. Maybe that's some utility scripts, maybe a section of your application
that you'll put in a gem using ruby-ffi. The functionality is still a long ways
off from either being practical, but don't underestimate how much I hate Ruby.

## Missing functionality

Many fundamental syntactic or semantic features of Ruby are not yet
implemented. If there are plans to address them, they are documented in the
issue tracker. There are some items for which there is explicitly no planned
support, however:

* Type hints/annotations -- the current type inference model relies on tracking
  method calls back to literal values. For library code that only ever is
  called in test or in client applications, this is insufficient.
* Exception handling -- the impedance mismatch between trapping runtime
  exceptions in Ruby and the comma-error pattern in Go is one of the largest
  differences between the two languages. It is large enough that I am avoiding
  implementing it entirely for now.  It may be possible but poses a number of
  difficulties that I believe can only be fully addressed by refactoring the
  source Ruby or refactoring the target Go.
* Metaprogramming -- I have no interest in building a Ruby runtime, which makes
  the extent of the metaprogramming that is realistic to support fairly small.
* Heterogenous arrays and hashes aren't on the menu. I do hope to support
  detection and generation of common interface types but in a fairly limited
  way.
* Hashes are translated directly into Go maps without any sort of shim type,
  which means that ordering guarantees provided by Ruby are not respected, and
  thus a number of Enumerable methods that depend on those guarantees are not
  supported.
* Features that, in my opinion, no one should be using anyway such as:
  * `BEGIN` and `END` blocks
  * `=begin`/`=end` comments
  * Interpolation of instance/class variables using `#@foo` instead of
    `#{@foo}` (did you even know you could do this??)
  * Global variables, including all the goofy automatically populated ones

Beyond large language features, thanos supports a very limited subset of
methods on core classes -- far too many to create issues for. Instead, they are
enumerated in the [docs folder](docs/missing_methods.md), generated using the
`report` command.

## How it works

The flow from bytes representing Ruby to bytes representing Go is as follows:

* A parser, generated with goyacc (see parser/ruby.y), consumes tokens using
  the lexer in parser/lexer.go and generates a parse tree using types
  implementing the `Node` interface in parser/ast.go.
* The resulting AST, stored on `*parser.Program`, then undergoes type inference
  by calling `Analyze()` on `*parser.Program`. This relies on:
  * the `parser.GetType` function and the `Type`, `SetType`, and `TargetType`
    methods on `Node`
  * the `types` package, which contains:
    * a `Type` interface
    * predefined implementations of the `Type` interface for Ruby primitive
      types and a small (but growing!) set of other classes from the Ruby
      standard library
    * facilities for generating new `Type`s for classes at parse time and for
      generating adapters from Ruby classes to Go functions when necessary
* The type-annotated AST is handed off to `compiler.Compile`, which translates
  `parser.Node`s into the appropriate analogs in the `go/ast` package. For
  method calls, this involves retrieving a `types.TransformAST` function, the
  return value of which supplies statements to prepend and an expression to
  substitute (more about this later). The resulting Go AST is then formatted and
  printed.

## How to add functionality to thanos

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
using Go's new generics:

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

I would say this is use case is overkill, and the handrolled AST is the right
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
for a single file, or just `go test ./compiler` to run them them all.

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

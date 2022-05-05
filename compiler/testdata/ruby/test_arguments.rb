def pos_and_kw(foo, bar: true)
  if bar
    puts foo
  end
end

pos_and_kw("x")
pos_and_kw("x", bar: false)

def all_kw(foo: "y", bar: true)
  if bar
    puts foo
  end
end

all_kw(bar: false)
all_kw(bar: false, foo: "z")

def defaults(foo = "x", bar = "y")
  "foo: #{foo}, bar: #{bar}"
end

defaults
defaults("z")
defaults("z", "a")

class Foo
  def initialize(foo = 10)
    @foo = foo
  end
end

Foo.new

def splat(a, *b, c: false)
  if c
    b[0]
  else
    a
  end
end

def double_splat(foo:, **bar)
  foo + bar[:baz]
end

splat(9, 2, 3)
splat(9, 2, c: true)
splat(9)
splat(9, *[1, 2])
splat(9, 5, *[1, 2])

double_splat(foo: 1, bar: 2, baz: 3)
double_splat(baz: 3, foo: 1)
double_splat(**{foo: 1, baz: 4})
hash_from_elsewhere = {foo: 1, baz: 4}
double_splat(**hash_from_elsewhere)

foo = [1, 2, 3]

a, *b = foo
c, d, *e = foo

syms = [:foo, :bar, :baz]

f = :quux, *syms
g, h, i = :quux, *syms
x, y, *z = :quux, *syms


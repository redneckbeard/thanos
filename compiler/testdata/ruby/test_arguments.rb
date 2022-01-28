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

h = {foo: "x", bar: "y"}

x = h.delete(:foo)

y = h.delete(:baz) do |k|
  "default for #{k}"
end

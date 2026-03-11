double = ->(x) { x * 2 }
puts double.call(5)

no_args = -> { "hello" }
puts no_args.call

add = ->(a, b) do
  a + b
end
puts add.call(3, 4)

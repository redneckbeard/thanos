gauntlet "LHS splat" do
  syms = [:foo, :bar, :baz, :quux]
  a, b, *c = syms
  puts a
  puts b
  c.each do |sym|
    puts sym
  end
end

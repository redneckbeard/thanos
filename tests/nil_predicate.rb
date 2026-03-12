gauntlet("nil? on value types") do
  x = "hello"
  puts x
  puts x.nil?
  y = 42
  puts y
  puts y.nil?
  z = [1, 2, 3]
  puts z.length
  puts z.nil?
end

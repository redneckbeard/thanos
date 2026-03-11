gauntlet("integer rational literal") do
  r = 3r
  puts r.numerator
  puts r.denominator
end

gauntlet("rational to_s") do
  r = 1r
  puts r.to_s
end

gauntlet("rational to_f") do
  r = 1r
  puts r.to_f
end

gauntlet("rational to_i") do
  r = 7r
  puts r.to_i
end

gauntlet("rational arithmetic add") do
  a = 1r
  b = 2r
  c = a + b
  puts c.to_s
end

gauntlet("rational arithmetic sub") do
  a = 3r
  b = 1r
  c = a - b
  puts c.to_s
end

gauntlet("rational arithmetic mul") do
  a = 2r
  b = 3r
  c = a * b
  puts c.to_s
end

gauntlet("rational arithmetic div") do
  a = 6r
  b = 3r
  c = a / b
  puts c.to_s
end

gauntlet("rational puts") do
  r = 3r
  puts r
end

gauntlet("imaginary float literal") do
  c = 2.5i
  puts c.imaginary
end

gauntlet("complex conjugate") do
  c = 2.5i
  puts c.conjugate.imaginary
end

gauntlet("complex to_f zero imaginary") do
  c = 0i
  puts c.to_f
end

gauntlet("complex to_i zero imaginary") do
  c = 0i
  puts c.to_i
end

gauntlet("tap on array") do
  arr = [1, 2, 3]
  result = arr.tap { |a| puts a.length }
  puts result[0]
  puts result[1]
  puts result[2]
end

gauntlet("tap on string") do
  s = "hello"
  result = s.tap { |x| puts x.length }
  puts result
end

gauntlet("tap on hash") do
  h = {"a" => 1, "b" => 2}
  result = h.tap { |x| puts x.length }
  puts result["a"]
end

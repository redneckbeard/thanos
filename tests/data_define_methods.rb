gauntlet("Data.define ==") do
  Point = Data.define(:x, :y)
  a = Point.new(1, 2)
  b = Point.new(1, 2)
  c = Point.new(1, 3)
  puts a == b
  puts a == c
  puts a.eql?(b)
  puts a.eql?(c)
end

gauntlet("Data.define deconstruct") do
  Point = Data.define(:x, :y)
  p = Point.new(1, 2)
  arr = p.deconstruct
  puts arr.length
  puts arr[0]
  puts arr[1]
end

gauntlet("Data.define to_h") do
  Point = Data.define(:x, :y)
  p = Point.new(1, 2)
  h = p.to_h
  puts h[:x]
  puts h[:y]
end

gauntlet("Data.define inspect") do
  Point = Data.define(:x, :y)
  p = Point.new(1, 2)
  puts p.inspect
end

gauntlet("Data.define members") do
  Point = Data.define(:x, :y)
  puts Point.members.length
  puts Point.members[0]
  puts Point.members[1]
end

gauntlet("Struct basic") do
  Point = Struct.new(:x, :y)
  p = Point.new(1, 2)
  puts p.x
  puts p.y
end

gauntlet("Struct setter") do
  Point = Struct.new(:x, :y)
  p = Point.new(3, 4)
  p.x = 10
  puts p.x
  puts p.y
end

gauntlet("Struct multiple instances") do
  Point = Struct.new(:x, :y)
  a = Point.new(1, 2)
  b = Point.new(3, 4)
  puts a.x + b.x
  puts a.y + b.y
end

gauntlet("Struct with strings") do
  Person = Struct.new(:name, :age)
  p = Person.new("Alice", 30)
  puts p.name
  puts p.age
end

gauntlet("Struct with block methods") do
  Point = Struct.new(:x, :y) do
    def to_s
      "#{x},#{y}"
    end
  end
  p = Point.new(3, 4)
  puts p.to_s
end

gauntlet("Struct with computed method") do
  Point = Struct.new(:x, :y) do
    def magnitude
      (x * x + y * y)
    end
  end
  p = Point.new(3, 4)
  puts p.magnitude
end

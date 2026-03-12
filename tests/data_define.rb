gauntlet("Data.define basic") do
  Point = Data.define(:x, :y)

  p = Point.new(1, 2)
  puts p.x
  puts p.y
end

gauntlet("Data.define with methods") do
  Measure = Data.define(:amount, :unit) do
    def to_s
      "#{amount} #{unit}"
    end
  end

  m = Measure.new(100, "km")
  puts m.to_s
end

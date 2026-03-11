module Geometry
  def self.pi
    3.14
  end

  class Circle
    def initialize(radius)
      @radius = radius
    end

    def area
      Geometry.pi * @radius * @radius
    end
  end
end

c = Geometry::Circle.new(10)
puts c.area

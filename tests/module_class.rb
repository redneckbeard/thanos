gauntlet("module with class") do
  module Animals
    class Dog
      def initialize(name)
        @name = name
      end

      def speak
        @name + " says woof"
      end
    end
  end

  d = Animals::Dog.new("Rex")
  puts d.speak
end

gauntlet("module with class and constant") do
  module Config
    Version = "1.0"

    class Settings
      Timeout = 30

      def initialize(name)
        @name = name
      end

      def describe
        @name + " v" + Version + " timeout=" + Timeout.to_s
      end
    end
  end

  s = Config::Settings.new("app")
  puts s.describe
end

gauntlet("module with class method and class") do
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
end

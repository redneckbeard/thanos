gauntlet("unused class skipped") do
  class Greeter
    def initialize(name)
      @name = name
    end

    def greet
      puts "Hello, #{@name}"
    end
  end

  class Farewell
    def initialize(name)
      @name = name
    end

    def farewell
      puts "Goodbye, #{@name}"
    end
  end

  # Only Greeter is used — Farewell is defined but never instantiated
  g = Greeter.new("world")
  g.greet
end

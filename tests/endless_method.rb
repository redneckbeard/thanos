gauntlet("endless method") do
  class Calculator
    attr_reader :name

    def initialize(name)
      @name = name
    end

    def double(x) = x * 2
    def triple(x) = x * 3
  end

  c = Calculator.new("calc")
  puts c.double(5)
  puts c.triple(4)
end

gauntlet("endless class method") do
  class Greeter
    def self.hello(name) = "Hello, #{name}!"
  end

  puts Greeter.hello("world")
end

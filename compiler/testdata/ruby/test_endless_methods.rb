class Calculator
  def double(x) = x * 2
  def triple(x) = x * 3
end

class Greeter
  def self.hello(name) = "Hello, #{name}!"
end

c = Calculator.new(0)
puts c.double(5)
puts c.triple(4)
puts Greeter.hello("world")

class Greeter
  class << self
    def hello(name)
      "Hello, #{name}!"
    end
  end
end

puts Greeter.hello("world")

gauntlet("singleton class basic") do
  class Greeter
    class << self
      def hello(name)
        "Hello, #{name}!"
      end

      def goodbye(name)
        "Goodbye, #{name}!"
      end
    end
  end

  puts Greeter.hello("world")
  puts Greeter.goodbye("world")
end

gauntlet("singleton class mixed with instance methods") do
  class Counter
    attr_reader :count

    def initialize
      @count = 0
    end

    def increment
      @count = @count + 1
    end

    class << self
      def description
        "A simple counter"
      end
    end
  end

  puts Counter.description
  c = Counter.new
  c.increment
  c.increment
  puts c.count
end

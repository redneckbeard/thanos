gauntlet("class << module constant") do
  module Utils
    def self.greet(name)
      puts "Hello, #{name}!"
    end
  end

  class << Utils
    def farewell(name)
      puts "Goodbye, #{name}!"
    end
  end

  Utils.greet("Alice")
  Utils.farewell("Bob")
end

gauntlet("class << nested module") do
  module Outer
    module Inner
      def self.original
        puts "original"
      end
    end
  end

  class << Outer::Inner
    def added
      puts "added via class <<"
    end
  end

  Outer::Inner.original
  Outer::Inner.added
end

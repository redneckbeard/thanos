gauntlet("nested module with scope") do
  module Outer
    module Inner
      def self.greet
        "hello from inner"
      end
    end
  end

  puts Outer::Inner.greet
end

gauntlet("nested module :: reopening") do
  module Animals
  end

  module Animals::Dogs
    def self.speak
      "woof"
    end
  end

  puts Animals::Dogs.speak
end

gauntlet("deeply nested module ::") do
  module A
    module B
    end
  end

  module A::B::C
    def self.value
      42
    end
  end

  puts A::B::C.value
end

gauntlet("module reopening add method") do
  module Outer2
    module Inner2
      def self.first
        "one"
      end
    end
  end

  module Outer2::Inner2
    def self.second
      "two"
    end
  end

  puts Outer2::Inner2.first
  puts Outer2::Inner2.second
end

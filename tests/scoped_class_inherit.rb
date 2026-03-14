gauntlet("scoped class with inheritance") do
  module Diff
    module LCS
    end
  end

  class Diff::LCS::Base
    def base_method
      "from base"
    end
  end

  class Diff::LCS::Child < Diff::LCS::Base
    def child_method
      base_method + " and child"
    end
  end

  c = Diff::LCS::Child.new
  puts c.child_method
end

gauntlet("same-class attr_reader bare name") do
  class Animal
    attr_reader :species

    def initialize(s)
      @species = s
    end

    def greet
      "hello #{species}"
    end
  end

  a = Animal.new("cat")
  puts a.greet
end

gauntlet("inherited attr_reader bare name") do
  class Pet
    attr_reader :kind

    def initialize(k)
      @kind = k
    end
  end

  class Puppy < Pet
    def describe
      "a " + kind
    end
  end

  p = Puppy.new("dog")
  puts p.describe
end

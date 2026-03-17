gauntlet("metaclass equality") do
  class Fruit
    def initialize(kind)
      @kind = kind
    end

    def same_class(other)
      self.class == other.class
    end
  end

  a = Fruit.new("apple")
  b = Fruit.new("banana")
  puts a.same_class(b)
end

gauntlet("class method called from instance method") do
  class Converter
    def self.rate
      100
    end

    def convert(amount)
      amount * Converter.rate
    end
  end

  c = Converter.new
  puts c.convert(5)
end

gauntlet("class method calls class method") do
  class MathHelper
    def self.double(x)
      x * 2
    end

    def self.quadruple(x)
      MathHelper.double(MathHelper.double(x))
    end
  end

  puts MathHelper.quadruple(3)
end

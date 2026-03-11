gauntlet("module method def self.x") do
  module MathUtils
    def self.double(x)
      x * 2
    end

    def self.add(a, b)
      a + b
    end
  end

  puts MathUtils.double(5)
  puts MathUtils.add(3, 7)
end

gauntlet("module method with string operations") do
  module StringUtils
    def self.shout(s)
      s.upcase + "!"
    end

    def self.whisper(s)
      s.downcase + "..."
    end
  end

  puts StringUtils.shout("hello")
  puts StringUtils.whisper("HELLO")
end

gauntlet("module method with array operations") do
  module ArrayUtils
    def self.sum_squares(arr)
      arr.map { |x| x * x }.sum
    end
  end

  puts ArrayUtils.sum_squares([1, 2, 3, 4])
end

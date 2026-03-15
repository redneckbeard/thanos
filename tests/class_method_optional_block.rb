gauntlet("class method with optional block") do
  class Transformer
    def self.transform(items, &blk)
      result = []
      items.each do |item|
        if block_given?
          result << yield(item)
        else
          result << item
        end
      end
      result
    end
  end

  r1 = Transformer.transform([1, 2, 3]) { |x| x * 10 }
  r1.each { |x| puts x }

  r2 = Transformer.transform([4, 5, 6])
  r2.each { |x| puts x }
end

gauntlet("class method block.call if block") do
  class Mapper
    def self.apply(values, &block)
      result = []
      values.each do |v|
        v = block.call(v) if block
        result << v
      end
      result
    end
  end

  r1 = Mapper.apply([10, 20, 30]) { |x| x + 1 }
  r1.each { |x| puts x }

  r2 = Mapper.apply([40, 50, 60])
  r2.each { |x| puts x }
end

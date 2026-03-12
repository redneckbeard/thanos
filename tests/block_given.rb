gauntlet("block_given? with block") do
  class Processor
    def process(x, &blk)
      if block_given?
        yield x
      else
        puts x
      end
    end
  end

  p = Processor.new
  p.process(42) { |v| puts v * 2 }
end

gauntlet("block_given? without block") do
  class Formatter
    def format(x, &blk)
      if block_given?
        yield x
      else
        puts x
      end
    end
  end

  f = Formatter.new
  f.format(10)
end

gauntlet("block_given? both paths") do
  class Handler
    def handle(x, &blk)
      if block_given?
        yield x
      else
        puts "default: #{x}"
      end
    end
  end

  h = Handler.new
  h.handle(5) { |v| puts "custom: #{v}" }
  h.handle(7)
end

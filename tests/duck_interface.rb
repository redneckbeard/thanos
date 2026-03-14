gauntlet("duck interface basic") do
  class DiffCallbacks
    def initialize(label)
      @label = label
    end

    def match(event)
      puts "#{@label} match: #{event}"
    end

    def discard(event)
      puts "#{@label} discard: #{event}"
    end
  end

  class SDiffCallbacks
    def initialize(label)
      @label = label
    end

    def match(event)
      puts "#{@label} MATCH: #{event}"
    end

    def discard(event)
      puts "#{@label} DISCARD: #{event}"
    end
  end

  def process(callbacks, items)
    items.each do |item|
      callbacks.match(item)
      callbacks.discard(item)
    end
  end

  process(DiffCallbacks.new("diff"), ["a", "b"])
  process(SDiffCallbacks.new("sdiff"), ["x", "y"])
end

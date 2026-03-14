gauntlet("respond_to? on duck interface") do
  class BasicCallbacks
    def match(event)
      puts "match: #{event}"
    end

    def discard(event)
      puts "discard: #{event}"
    end
  end

  class FullCallbacks
    def match(event)
      puts "MATCH: #{event}"
    end

    def discard(event)
      puts "DISCARD: #{event}"
    end

    def change(event)
      puts "CHANGE: #{event}"
    end
  end

  def process(callbacks, items)
    items.each do |item|
      callbacks.match(item)
      callbacks.discard(item)
      if callbacks.respond_to?(:change)
        callbacks.change(item)
      end
    end
  end

  process(BasicCallbacks.new, ["a"])
  process(FullCallbacks.new, ["x"])
end

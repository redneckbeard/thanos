gauntlet("block_given? yield with value") do
  def process(items)
    results = []
    items.each do |item|
      item = yield item if block_given?
      results << item
    end
    results
  end

  r1 = process(["a", "b"]) { |x| x.upcase }
  r1.each { |x| puts x }

  r2 = process(["c", "d"])
  r2.each { |x| puts x }
end

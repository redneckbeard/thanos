gauntlet("block_given? with yield value") do
  def process(items, label)
    results = []
    items.each do |item|
      item = yield item if block_given?
      results << "#{label}: #{item}"
    end
    results
  end

  r1 = process(["a", "b"], "with") { |x| x.upcase }
  r1.each { |x| puts x }

  r2 = process(["c", "d"], "without")
  r2.each { |x| puts x }
end

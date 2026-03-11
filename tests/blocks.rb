gauntlet("user-defined block method") do
  def logging(arr, &blk)
    arr.each do |n|
      blk.call(n)
    end
  end

  logging([1,2,3,4,5]) do |x|
    puts x
  end
end

gauntlet("block with return value") do
  def apply(x, &blk)
    blk.call(x)
  end

  puts apply(5) { |n| n * n }
end

gauntlet("multiple method calls") do
  def double(x)
    x * 2
  end

  def add_one(x)
    x + 1
  end

  puts add_one(double(3))
end

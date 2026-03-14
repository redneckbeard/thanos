gauntlet("block_given? implicit block") do
  def process(x)
    if block_given?
      yield x
    else
      puts x
    end
  end

  process(42) { |v| puts v * 2 }
  process(10)
end

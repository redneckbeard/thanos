gauntlet("Int#abs") do
  #TODO bug, parens required
  puts(-10.abs)
  puts(10.abs)
end

gauntlet("Int#negative?") do
  #TODO bug, parens required
  puts(-10.negative?)
  puts(0.negative?)
  puts(10.negative?)
end

gauntlet("Int#positive?") do
  #TODO bug, parens required
  puts(-10.positive?)
  puts(0.positive?)
  puts(10.positive?)
end

gauntlet("Int#zero?") do
  #TODO bug, parens required
  puts(-10.zero?)
  puts(0.zero?)
  puts(10.zero?)
end

gauntlet("Int#times") do
  10.times do |i|
    puts i
  end
end

gauntlet("Int#upto") do
  10.upto(20) do |i|
    puts i
  end
end

gauntlet("Int#downto") do
  20.downto(10) do |i|
    puts i
  end
end

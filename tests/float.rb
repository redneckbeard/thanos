gauntlet("Float#negative?") do
  #TODO bug, parens required
  puts(-10.0.negative?)
  puts(0.0.negative?)
  puts(10.0.negative?)
end

gauntlet("Float#positive?") do
  #TODO bug, parens required
  puts(-10.0.positive?)
  puts(0.0.positive?)
  puts(10.0.positive?)
end

gauntlet("Float#zero?") do
  #TODO bug, parens required
  puts(-10.0.zero?)
  puts(0.0.zero?)
  puts(10.0.zero?)
end

gauntlet("Float#abs") do
  puts(-10.0.abs.positive?)
end

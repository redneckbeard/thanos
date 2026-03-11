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

gauntlet("Float#ceil") do
  puts(3.2.ceil)
  puts(3.8.ceil)
end

gauntlet("Float#floor") do
  puts(3.2.floor)
  puts(3.8.floor)
end

gauntlet("Float#round") do
  puts(3.2.round)
  puts(3.5.round)
  puts(3.8.round)
end

gauntlet("Float#to_i") do
  x = 3.7
  puts x.to_i
end

gauntlet("Float#abs") do
  puts(-3.14.abs)
  puts(3.14.abs)
end

gauntlet("Float#nan?") do
  puts(0.0.nan?)
end

gauntlet("Float#between?") do
  puts 3.5.between?(1.0, 5.0)
  puts 3.5.between?(4.0, 5.0)
end

gauntlet("Float#clamp") do
  puts 3.5.clamp(1.0, 5.0)
  puts 0.5.clamp(1.0, 5.0)
  puts 7.5.clamp(1.0, 5.0)
end

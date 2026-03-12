gauntlet("Integer#abs") do
  puts (-5).abs
  puts 3.abs
  puts 0.abs
end

gauntlet("Integer#succ") do
  puts 5.succ
  puts (-1).succ
end

gauntlet("Integer#pred") do
  puts 5.pred
  puts 0.pred
end

gauntlet("Integer#zero?") do
  puts 0.zero?
  puts 5.zero?
end

gauntlet("Integer#nonzero?") do
  puts 5.nonzero? ? "yes" : "no"
  puts 0.nonzero? ? "yes" : "no"
end

gauntlet("Integer#positive? and negative?") do
  puts 5.positive?
  puts (-3).positive?
  puts 0.positive?
  puts (-3).negative?
  puts 5.negative?
end

gauntlet("Integer#even? and odd?") do
  puts 4.even?
  puts 5.even?
  puts 4.odd?
  puts 5.odd?
end

gauntlet("Integer ceil floor round are identity") do
  puts 5.ceil
  puts 5.floor
  puts 5.round
end

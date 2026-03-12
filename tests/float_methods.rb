gauntlet("Float#truncate") do
  puts 3.7.truncate
  puts (-2.3).truncate
end

gauntlet("Float#modulo") do
  puts 10.5.modulo(3.0)
end

gauntlet("Float#positive? and negative?") do
  puts 3.14.positive?
  puts (-2.7).positive?
  puts (-2.7).negative?
  puts 3.14.negative?
end

gauntlet("Float#zero?") do
  puts 0.0.zero?
  puts 3.14.zero?
end

gauntlet("Float#to_f is identity") do
  puts 3.14.to_f
end

gauntlet("Float#abs") do
  puts (-3.14).abs
  puts 2.7.abs
end

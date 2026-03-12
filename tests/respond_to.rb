gauntlet("respond_to? on string") do
  puts "hello".respond_to?(:upcase)
  puts "hello".respond_to?(:nonexistent)
end

gauntlet("respond_to? on array") do
  puts [1, 2, 3].respond_to?(:length)
  puts [1, 2, 3].respond_to?(:foo)
end

gauntlet("respond_to? on integer") do
  puts 42.respond_to?(:to_s)
  puts 42.respond_to?(:bar)
end

gauntlet("respond_to? conditional") do
  s = "hello"
  if s.respond_to?(:upcase)
    puts s.upcase
  else
    puts "no upcase"
  end
end

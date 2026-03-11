gauntlet("range assigned to a local used in a case statement") do
  loc = 1..5

  10.times do |i|
    case i
    when loc
      puts "#{i} in range"
    else
      puts "#{i} out of range"
    end
  end
end

gauntlet("range of strings in a case statement") do
  ["foo", "bar", "baz"].each do |str|
    case str
    when "bar".."baz"
      puts "it's a hit!"
    else
      puts "it's a miss"
    end
  end
end

gauntlet("range each") do
  (1..5).each do |i|
    puts i
  end
end

gauntlet("range each exclusive") do
  (1...5).each do |i|
    puts i
  end
end

gauntlet("range to_a") do
  puts (1..5).to_a.length
end

gauntlet("range map") do
  puts (1..5).map { |i| i * 2 }.join(", ")
end

gauntlet("range select") do
  puts (1..5).select { |i| i > 3 }.join(", ")
end

gauntlet("range include") do
  puts (1..10).include?(5)
  puts (1..10).include?(11)
end

gauntlet("range size") do
  puts (1..10).size
  puts (1...10).size
end

gauntlet("range min max") do
  puts (3..7).min
  puts (3..7).max
end

gauntlet("range first last") do
  puts (1..10).first
  puts (1..10).last
end

gauntlet("range sum") do
  puts (1..100).sum
end

gauntlet("range any? none?") do
  puts (1..10).any? { |i| i > 5 }
  puts (1..10).none? { |i| i > 20 }
end

gauntlet("range reduce") do
  puts (1..5).reduce(0) { |sum, i| sum + i }
end

gauntlet("range each_with_index") do
  (10..13).each_with_index do |val, idx|
    puts "#{idx}: #{val}"
  end
end

gauntlet("range reject") do
  puts (1..10).reject { |i| i % 2 == 0 }.join(", ")
end

gauntlet("range find") do
  puts (1..100).find { |i| i > 50 }
end

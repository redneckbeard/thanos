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

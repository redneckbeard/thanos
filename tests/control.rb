gauntlet("break") do
  10.times do |i|
    puts i
    if i > 5
      break
    end
  end
end

gauntlet("next") do
  10.times do |i|
    puts i
    if i % 2 == 0
      next
    end
  end
end

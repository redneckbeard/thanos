gauntlet("loop do break") do
  i = 0
  loop do
    break if i >= 5
    i += 1
  end
  puts i
end

gauntlet("case array pattern") do
  a = true
  b = false
  case [a, b]
  when [true, true]
    puts "both"
  when [true, false]
    puts "first only"
  when [false, true]
    puts "second only"
  else
    puts "neither"
  end

  # Test with expressions
  x = 3
  y = 5
  case [(x < 4), (y < 4)]
  when [true, true]
    puts "both small"
  when [true, false]
    puts "x small"
  when [false, true]
    puts "y small"
  else
    puts "both big"
  end
end

gauntlet("case in pattern") do
  # Simple destructuring
  arr = ["add", "5", "hello"]
  case arr
  in [action, position, element]
    puts action
    puts position
    puts element
  else
    puts "no match"
  end

  # Non-matching length
  short = ["x"]
  case short
  in [a, b, c]
    puts "matched"
  else
    puts "too short"
  end
end

gauntlet("pattern match array destructure") do
  arr = [1, 2, 3]
  case arr
  in [a, b, c]
    puts a
    puts b
    puts c
  end
end

gauntlet("pattern match multiple clauses") do
  arr = [10, 20]
  case arr
  in [a, b, c]
    puts "three"
  in [a, b]
    puts a
    puts b
  end
end

gauntlet("pattern match with else") do
  arr = [1]
  case arr
  in [a, b, c]
    puts "three"
  in [a, b]
    puts "two"
  else
    puts "other"
  end
end

gauntlet("pattern match wildcard") do
  arr = [1, 2, 3]
  case arr
  in [_, b, _]
    puts b
  end
end

gauntlet("pattern match value") do
  arr = [1, 2, 3]
  case arr
  in [1, b, 3]
    puts b
  end
end

gauntlet("pattern match nested arrays") do
  arr = [[1, 2], [3, 4]]
  case arr
  in [[a, b], [c, d]]
    puts a
    puts b
    puts c
    puts d
  end
end

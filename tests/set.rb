gauntlet("Set#union") do
  require "set"
  arr = []
  Set[1, 2, 3].union(Set[2, 4, 5]).each do |x|
    arr << x
  end
  arr.sort!.each do |x|
    puts x
  end
end

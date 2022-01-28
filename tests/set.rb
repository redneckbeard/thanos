gauntlet("Set#union") do
  require "set"
  Set[1, 2, 3].union(Set[2, 4, 5]).each do |x|
    puts x
  end
end

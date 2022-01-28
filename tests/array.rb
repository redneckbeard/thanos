gauntlet("join") do
  puts([1, 2, 3].join(" + "))
  puts(["foo", "bar", "baz"].join(" and "))
end

gauntlet("take") do
  [1,2,3,4,5].take(3).each do |x|
    puts x
  end
end

gauntlet("drop") do
  [1,2,3,4,5].drop(3).each do |x|
    puts x
  end
end

gauntlet("values_at") do
  [1,2,3,4,5].values_at(2, 4).each do |x|
    puts x
  end
end

gauntlet("unshift") do
  orig = [1,2,3,4,5]
  orig.unshift(2, 4).each { |x| puts x }
end

gauntlet("arr[x..]") do
  orig = [1,2,3,4,5]
  orig[3..].each { |x| puts x }
end

gauntlet("arr[x...]") do
  orig = [1,2,3,4,5]
  orig[3...].each { |x| puts x }
end

gauntlet("arr[x..y]") do
  orig = [1,2,3,4,5]
  orig[1..4].each { |x| puts x }
end

gauntlet("arr[x...y]") do
  orig = [1,2,3,4,5]
  orig[1...4].each { |x| puts x }
end

gauntlet("arr[x...-y]") do
  orig = [1,2,3,4,5]
  orig[1...-2].each { |x| puts x }
end

gauntlet("arr[x..-y]") do
  orig = [1,2,3,4,5]
  orig[1..-2].each { |x| puts x }
end

gauntlet("arr[x..y] with x var") do
  orig = [1,2,3,4,5]
   foo = 1
  orig[foo..-2].each { |x| puts x }
end

gauntlet("arr[x..y] with y var") do
  orig = [1,2,3,4,5]
  foo = 3
  orig[1..foo].each { |x| puts x }
end

gauntlet("arr[x...y] with y var") do
  orig = [1,2,3,4,5]
  foo = 3
  orig[1...foo].each { |x| puts x }
end

gauntlet("arr[x..y] with -y var") do
  orig = [1,2,3,4,5]
  foo = -2
  orig[1..foo].each { |x| puts x }
end

gauntlet("arr[x...y] with -y var") do
  orig = [1,2,3,4,5]
  foo = -2
  orig[1...foo].each { |x| puts x }
end

gauntlet("include?") do
  orig = [1,2,3,4,5]
  puts orig.include?(4)
  puts orig.include?(7)
end

gauntlet("Array#-") do
  set = [1, 2, 3, 4] - [2, 3] # this should be ([] - []).each but fails because of a parser bug
  set.each do |i|
    puts i
  end
end

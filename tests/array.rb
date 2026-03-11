gauntlet("Array#first with no arguments") do
  arr = [1, 2, 3, 4, 5]
  puts arr.first
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

gauntlet("Array#+") do
  ([1, 2] + [3, 4]).each do |i|
    puts i
  end
end

gauntlet("Array#-") do
  ([1, 2, 3, 4] - [2, 3]).each do |i|
    puts i
  end
end

gauntlet("uniq") do
  [1, 2, 2, 3, 4, 4, 4].uniq.each do |i|
    puts i
  end
end

gauntlet("Array#first with argument") do
  arr = [1, 2, 3, 4, 5]
  puts arr.first(3).join(",")
  puts arr.first(0).join(",")
  puts arr.first(10).join(",")
end

gauntlet("Empty array type inference with join") do
  empty = []
  puts empty.join(",")
end

gauntlet("Empty array type inference with first and join") do
  empty = []
  result = empty.first(2)
  puts result.join(",")
end

gauntlet("take") do
  [1,2,3,4,5].take(3).each do |x|
    puts x
  end
end

gauntlet("join") do
  puts [1, 2, 3].join(" + ")
  puts ["foo", "bar", "baz"].join(" and ")
end

gauntlet("Array#last with no arguments") do
  arr = [1, 2, 3, 4, 5]
  puts arr.last
end

gauntlet("Array#last with argument") do
  arr = [1, 2, 3, 4, 5]
  puts arr.last(3).join(",")
  puts arr.last(0).join(",")
  puts arr.last(10).join(",")
end

gauntlet("Array#empty?") do
  arr = [1, 2, 3]
  puts arr.empty?
  empty_arr = []
  puts empty_arr.empty?
end

gauntlet("Array#push") do
  arr = [1, 2, 3]
  arr.push(4, 5)
  puts arr.join(",")
end

gauntlet("Array#pop with no arguments") do
  arr = [1, 2, 3, 4, 5]
  popped = arr.pop
  puts popped
  puts arr.join(",")
end

gauntlet("Array#pop with argument") do
  arr = [1, 2, 3, 4, 5]
  popped = arr.pop(2)
  puts popped.join(",")
  puts arr.join(",")
end

gauntlet("Array#push with empty array type inference") do
  empty_ints = []
  empty_ints.push(42)
  empty_ints.push(100)
  puts empty_ints.length
end

gauntlet("Array#map") do
  [1, 2, 3, 4].map { |x| x * x }.each do |x|
    puts x
  end
end

gauntlet("Array#select") do
  [1, 2, 3, 4, 5, 6].select { |x| x % 2 == 0 }.each do |x|
    puts x
  end
end

gauntlet("Array#reduce") do
  puts [1, 2, 3, 4, 5].reduce(0) { |acc, x| acc + x }
end

gauntlet("Array#length") do
  puts [1, 2, 3].length
  puts [1, 2, 3].size
end

gauntlet("Array#sort!") do
  arr = [3, 1, 4, 1, 5, 9, 2, 6]
  arr.sort!
  arr.each { |x| puts x }
end

gauntlet("Array#reject") do
  [1, 2, 3, 4, 5, 6].reject { |x| x % 2 == 0 }.each do |x|
    puts x
  end
end

gauntlet("Array#collect") do
  [1, 2, 3].collect { |x| x + 10 }.each { |x| puts x }
end

gauntlet("Array#inject") do
  puts [1, 2, 3, 4].inject(0) { |acc, x| acc + x }
end

gauntlet("Array#each chaining") do
  [1, 2, 3].map { |x| x * 2 }.select { |x| x > 2 }.each do |x|
    puts x
  end
end

gauntlet("multiple assignment") do
  a, b, c = 1, 2, 3
  puts a
  puts b
  puts c
end

gauntlet("swap assignment") do
  a = 10
  b = 20
  a, b = b, a
  puts a
  puts b
end

gauntlet("Array#count") do
  puts [1, 2, 3, 4, 5].count
end

gauntlet("Array#reverse") do
  [1, 2, 3, 4, 5].reverse.each { |x| puts x }
end

gauntlet("Array#each_with_index") do
  ["a", "b", "c"].each_with_index do |val, idx|
    puts "#{idx}: #{val}"
  end
end

gauntlet("Array#min") do
  puts [3, 1, 4, 1, 5, 9].min
end

gauntlet("Array#max") do
  puts [3, 1, 4, 1, 5, 9].max
end

gauntlet("Array#sum") do
  puts [1, 2, 3, 4, 5].sum
end

gauntlet("Array#any?") do
  puts [1, 2, 3, 4, 5].any? { |x| x > 3 }
  puts [1, 2, 3, 4, 5].any? { |x| x > 10 }
end

gauntlet("Array#none?") do
  puts [1, 2, 3].none? { |x| x > 5 }
  puts [1, 2, 3].none? { |x| x > 2 }
end

gauntlet("Array#all?") do
  puts [2, 4, 6, 8].all? { |x| x % 2 == 0 }
  puts [2, 4, 5, 8].all? { |x| x % 2 == 0 }
end

gauntlet("Array#each_with_object") do
  result = [1, 2, 3].each_with_object([]) do |x, arr|
    arr << x * 2
  end
  result.each { |x| puts x }
end

gauntlet("Array#index") do
  arr = [10, 20, 30, 40, 50]
  puts arr.index(30)
  puts arr.index(99)
end

gauntlet("Array#concat") do
  arr = [1, 2, 3]
  arr.concat([4, 5])
  arr.each { |x| puts x }
end

gauntlet("Array#clear") do
  arr = [1, 2, 3]
  arr.clear
  puts arr.length
end

gauntlet("Array#flat_map") do
  [[1, 2], [3, 4], [5, 6]].flat_map { |arr| arr }.each { |x| puts x }
end

gauntlet("Array#flatten") do
  [[1, 2], [3, 4], [5, 6]].flatten.each { |x| puts x }
end

gauntlet("Array#compact") do
  arr = ["a", nil, "b", nil, "c"]
  puts arr.compact.join(", ")
end

gauntlet("Array#compact with integers") do
  arr = [1, nil, 2, nil, 3]
  result = arr.compact
  result.each { |x| puts x }
  puts result.length
end

gauntlet("nil || default") do
  arr = ["a", nil, "b"]
  x = arr[1] || "default"
  puts x
  y = arr[0] || "fallback"
  puts y
end

gauntlet("nil ||= default") do
  arr = [1, nil, 3]
  x = arr[1]
  x ||= 99
  puts x
  y = arr[0]
  y ||= 99
  puts y
end

gauntlet("safe navigation &.") do
  arr = ["hello", nil, "world"]
  x = arr[0]&.upcase
  puts x || "nil"
  y = arr[1]&.upcase
  puts y || "nil"
end

gauntlet("Array#find") do
  result = [1, 2, 3, 4, 5].find { |x| x > 3 }
  puts result || -1
  no_match = [1, 2, 3].find { |x| x > 10 }
  puts no_match || -1
end

gauntlet("Array#detect") do
  result = ["apple", "banana", "cherry"].detect { |s| s.start_with?("b") }
  puts result || "none"
end

gauntlet("Array#sort") do
  [3, 1, 4, 1, 5, 9].sort.each { |x| puts x }
end

gauntlet("Array#sort_by") do
  ["banana", "fig", "cherry", "apple"].sort_by { |s| s.size }.each { |s| puts s }
end

gauntlet("Array#group_by") do
  grouped = [1, 2, 3, 4, 5, 6].group_by { |x| x % 3 }
  grouped.keys.each do |k|
    grouped[k].each { |v| puts "#{k}: #{v}" }
  end
end

gauntlet("Array#partition") do
  evens, odds = [1, 2, 3, 4, 5, 6].partition { |x| x % 2 == 0 }
  evens.each { |x| puts x }
  puts "---"
  odds.each { |x| puts x }
end

gauntlet("Array#each_slice") do
  [1, 2, 3, 4, 5, 6, 7].each_slice(3) do |chunk|
    puts chunk.join(",")
  end
end

gauntlet("Array#min_by") do
  puts ["apple", "fig", "banana"].min_by { |s| s.length }
end

gauntlet("Array#max_by") do
  puts ["apple", "fig", "banana"].max_by { |s| s.length }
end

gauntlet("Array#delete") do
  arr = [1, 2, 3, 2, 4]
  arr.delete(2)
  arr.each { |x| puts x }
end

gauntlet("Array#each_cons") do
  [1, 2, 3, 4, 5].each_cons(3) do |group|
    puts group.join(",")
  end
end

gauntlet("Array#rotate") do
  [1, 2, 3, 4, 5].rotate(2).each { |x| puts x }
end

gauntlet("Array#intersection") do
  ([1, 2, 3, 4] & [2, 4, 6]).each { |x| puts x }
end

gauntlet("Array#union") do
  ([1, 2, 3] | [2, 3, 4]).each { |x| puts x }
end

gauntlet("Array#flatten chained") do
  [[1, 2], [3, 4]].flatten.each { |x| puts x }
end

gauntlet("negative array indexing") do
  arr = [10, 20, 30, 40, 50]
  puts arr[-1]
  puts arr[-2]
  puts arr[-5]
end

gauntlet("negative array index assignment") do
  arr = [1, 2, 3, 4, 5]
  arr[-1] = 99
  puts arr[-1]
  arr[-3] = 77
  arr.each { |x| puts x }
end

gauntlet("multiline method chain") do
  result = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    .select { |x| x > 3 }
    .map { |x| x * 10 }
  result.each { |x| puts x }
end

gauntlet("Array#delete_at") do
  arr = [10, 20, 30, 40, 50]
  val = arr.delete_at(2)
  puts val
  arr.each { |x| puts x }
end


gauntlet("Array#fetch with default") do
  arr = [10, 20, 30]
  puts arr.fetch(1, 99)
  puts arr.fetch(5, 99)
end

gauntlet("Array#each_index") do
  arr = ["a", "b", "c"]
  arr.each_index do |i|
    puts i
  end
end

gauntlet("Array#dig") do
  arr = [[1, 2], [3, 4], [5, 6]]
  puts arr.dig(1, 0)
  puts arr.dig(2, 1)
end

gauntlet("Array#one?") do
  puts [1, 2, 3].one? { |x| x > 2 }
  puts [1, 2, 3].one? { |x| x > 1 }
  puts [1, 2, 3].one? { |x| x > 10 }
end

gauntlet("Array#count with block") do
  puts [1, 2, 3, 4, 5].count { |x| x > 3 }
  puts [1, 2, 3, 4, 5].count { |x| x % 2 == 0 }
end

gauntlet("Array#find_index") do
  arr = [10, 20, 30, 40]
  puts arr.find_index { |x| x > 25 }
  puts arr.find_index { |x| x > 100 }
end

gauntlet("Array#reverse_each") do
  arr = [1, 2, 3, 4, 5]
  arr.reverse_each { |x| puts x }
end

gauntlet("Array#map!") do
  arr = [1, 2, 3, 4, 5]
  arr.map! { |x| x * 10 }
  arr.each { |x| puts x }
end

gauntlet("Array#collect!") do
  arr = ["foo", "bar", "baz"]
  arr.collect! { |s| s.upcase }
  arr.each { |s| puts s }
end

gauntlet("Array#map! type change") do
  arr = [1, 2, 3]
  arr.map! { |x| x.to_s }
  arr.each { |s| puts s }
end

gauntlet("Array#select!") do
  arr = [1, 2, 3, 4, 5, 6, 7, 8]
  arr.select! { |x| x % 2 == 0 }
  arr.each { |x| puts x }
end

gauntlet("Array#filter!") do
  arr = [10, 20, 30, 40, 50]
  arr.filter! { |x| x > 25 }
  arr.each { |x| puts x }
end

gauntlet("Array#reject!") do
  arr = [1, 2, 3, 4, 5, 6]
  arr.reject! { |x| x % 3 == 0 }
  arr.each { |x| puts x }
end

gauntlet("Array#delete_if") do
  arr = [1, 2, 3, 4, 5]
  arr.delete_if { |x| x < 3 }
  arr.each { |x| puts x }
end

gauntlet("Array#reverse!") do
  arr = [1, 2, 3, 4, 5]
  arr.reverse!
  arr.each { |x| puts x }
end

gauntlet("Array#uniq!") do
  arr = [1, 2, 2, 3, 3, 3, 4]
  arr.uniq!
  arr.each { |x| puts x }
end

gauntlet("Array#sort_by!") do
  arr = ["banana", "fig", "apple", "date"]
  arr.sort_by! { |s| s.length }
  arr.each { |s| puts s }
end

gauntlet("Array#delete") do
  arr = [1, 2, 3, 2, 4, 2]
  puts arr.delete(2)
  arr.each { |x| puts x }
end

gauntlet("Array#shift") do
  arr = [10, 20, 30, 40]
  puts arr.shift
  arr.each { |x| puts x }
end

gauntlet("Array#insert") do
  arr = [1, 2, 3]
  arr.insert(1, 99)
  arr.each { |x| puts x }
end

gauntlet("Array#zip") do
  a = [1, 2, 3]
  b = [4, 5, 6]
  a.zip(b).each do |pair|
    puts pair.join(",")
  end
end

gauntlet("Array#take_while") do
  arr = [1, 2, 3, 4, 5]
  result = arr.take_while { |x| x < 4 }
  result.each { |x| puts x }
end

gauntlet("Array#drop_while") do
  arr = [1, 2, 3, 4, 5]
  result = arr.drop_while { |x| x < 4 }
  result.each { |x| puts x }
end

gauntlet("Array#fill") do
  arr = [1, 2, 3, 4, 5]
  arr.fill(0)
  arr.each { |x| puts x }
end

gauntlet("Array#combination") do
  [1, 2, 3, 4].combination(2).each do |combo|
    puts combo.join(",")
  end
end

gauntlet("Array#permutation") do
  [1, 2, 3].permutation(2).each do |perm|
    puts perm.join(",")
  end
end

gauntlet("Array#product") do
  [1, 2].product([3, 4]).each do |pair|
    puts pair.join(",")
  end
end

gauntlet("Array#rindex") do
  arr = [1, 2, 3, 2, 1]
  puts arr.rindex(2)
end

gauntlet("Array#transpose") do
  arr = [[1, 2], [3, 4], [5, 6]]
  arr.transpose.each do |row|
    puts row.join(",")
  end
end

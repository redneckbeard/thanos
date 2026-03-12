gauntlet("numbered params simple") do
  arr = [1, 2, 3]
  arr.each { puts _1 }
end

gauntlet("numbered params with operation") do
  arr = [10, 20, 30]
  result = arr.map { _1 * 2 }
  result.each { puts _1 }
end

gauntlet("numbered params two args") do
  h = {"a" => 1, "b" => 2, "c" => 3}
  h.each { puts "#{_1}: #{_2}" }
end

gauntlet("numbered params select") do
  arr = [1, 2, 3, 4, 5, 6]
  evens = arr.select { _1 % 2 == 0 }
  evens.each { puts _1 }
end

arr = [1, 2, 3]
arr.each { puts _1 }
result = arr.map { _1 * 2 }

h = {"a" => 1, "b" => 2}
h.each { puts "#{_1}: #{_2}" }

counts = Hash.new(0)
counts["a"] += 1
counts["b"] += 2
puts counts["a"]
puts counts["c"]
puts counts.keys.length

gauntlet("Hash#delete") do
  puts({:foo => "x", :bar => "y"}.delete(:foo))
end

gauntlet("Hash#delete as statement") do
  h = {foo: 1, bar: 2, baz: 3}
  h.delete(:foo)
  h.keys.each { |k| puts "#{k}: #{h[k]}" }
end

gauntlet("Hash#delete (with block)") do
  result = {:foo => "x", :bar => "y"}.delete(:baz) do |k|
    "default: #{k}"
  end
  puts result
end

gauntlet("Hash#delete_if") do
  h = {foo: 1, bar: 2, baz: 3, quux: 4}
  smaller = h.delete_if do |k, v|
    v > 2
  end
  # we have the keys removed
  h.keys.each do |k|
    puts "#{k}: #{h[k]}"
  end
  puts smaller.length == h.length
end

gauntlet("Hash#values") do
  h = {foo: 1, bar: 2, baz: 3, quux: 4}
  if h.has_value?(3)
    h.values.each do |v|
      puts v
    end
  end
end

gauntlet("Hash#keys") do
  h = {foo: 1, bar: 2, baz: 3, quux: 4}
  if h.has_key?(:foo)
    h.keys.each do |k|
      puts k
    end
  end
end

gauntlet("Hash#empty?") do
  puts({}.empty?)
  puts({foo: 1}.empty?)
end

gauntlet("Hash#length") do
  puts({foo: 1, bar: 2, baz: 3}.length)
  puts({foo: 1, bar: 2, baz: 3}.size)
end

gauntlet("Hash#each") do
  h = {a: 1, b: 2, c: 3}
  keys = []
  vals = []
  h.each do |k, v|
    keys << k
    vals << v
  end
  keys.each { |k| puts k }
  vals.each { |v| puts v }
end

gauntlet("Hash#filter") do
  h = {a: 1, b: 2, c: 3, d: 4}
  result = h.filter do |k, v|
    v > 2
  end
  result.keys.each { |k| puts k }
end

gauntlet("Hash#each_key") do
  h = {a: 1, b: 2, c: 3}
  keys = []
  h.each_key do |k|
    keys << k
  end
  keys.each { |k| puts k }
end

gauntlet("Hash#each_value") do
  h = {a: 1, b: 2, c: 3}
  vals = []
  h.each_value do |v|
    vals << v
  end
  vals.each { |v| puts v }
end

gauntlet("Hash#has_key?") do
  h = {foo: 1, bar: 2}
  puts h.has_key?(:foo)
  puts h.has_key?(:baz)
end

gauntlet("Hash#has_value?") do
  h = {foo: 1, bar: 2}
  puts h.has_value?(1)
  puts h.has_value?(99)
end

gauntlet("Hash#clear") do
  h = {foo: 1, bar: 2}
  h.clear
  puts h.empty?
end

gauntlet("Hash#merge") do
  h1 = {a: 1, b: 2}
  h2 = {b: 3, c: 4}
  merged = h1.merge(h2)
  merged.keys.each { |k| puts "#{k}: #{merged[k]}" }
end

gauntlet("Hash#reject") do
  h = {a: 1, b: 2, c: 3, d: 4}
  result = h.reject do |k, v|
    v > 2
  end
  result.keys.each { |k| puts k }
end

gauntlet("Hash#count") do
  puts({foo: 1, bar: 2, baz: 3}.count)
end

gauntlet("Hash#select") do
  h = {a: 1, b: 2, c: 3, d: 4}
  result = h.select do |k, v|
    v > 2
  end
  result.keys.each { |k| puts k }
end

gauntlet("Hash#any?") do
  h = {a: 1, b: 2, c: 3}
  puts h.any? { |k, v| v > 2 }
  puts h.any? { |k, v| v > 10 }
end

gauntlet("Hash#map") do
  h = {a: 1, b: 2, c: 3}
  result = h.map { |k, v| "#{k}=#{v}" }
  result.each { |s| puts s }
end

gauntlet("Hash#transform_values") do
  h = {a: 1, b: 2, c: 3}
  doubled = h.transform_values { |v| v * 2 }
  doubled.keys.each { |k| puts "#{k}: #{doubled[k]}" }
end

gauntlet("Hash#include?") do
  h = {foo: 1, bar: 2}
  puts h.include?(:foo)
  puts h.include?(:baz)
end

gauntlet("Hash#each_with_object") do
  h = {a: 1, b: 2, c: 3}
  result = h.each_with_object([]) do |(k, v), arr|
    arr << "#{k}=#{v}"
  end
  result.each { |s| puts s }
end

gauntlet("Hash#fetch with default") do
  h = {a: 1, b: 2, c: 3}
  puts h.fetch(:a, 99)
  puts h.fetch(:z, 99)
end

gauntlet("Hash#fetch without default") do
  h = {a: 1, b: 2, c: 3}
  puts h.fetch(:b)
end

gauntlet("h[key] || default") do
  h = {a: 1, b: 2, c: 3}
  x = h[:a] || 99
  puts x
  y = h[:z] || 99
  puts y
end

gauntlet("Hash#invert") do
  h = {a: 1, b: 2, c: 3}
  inv = h.invert
  puts inv[1]
  puts inv[2]
  puts inv[3]
end

gauntlet("Hash#dig single key") do
  h = {a: 1, b: 2, c: 3}
  puts h.dig(:b)
end

gauntlet("Hash#dig nested") do
  h = {a: {x: 1, y: 2}, b: {x: 3, y: 4}}
  puts h.dig(:a, :x)
  puts h.dig(:b, :y)
end

gauntlet("Hash#key?") do
  h = {a: 1, b: 2}
  puts h.key?(:a)
  puts h.key?(:z)
end

gauntlet("Hash#value?") do
  h = {a: 1, b: 2}
  puts h.value?(2)
  puts h.value?(99)
end

gauntlet("Hash.new with default value") do
  counts = Hash.new(0)
  counts["a"] += 1
  counts["b"] += 2
  counts["a"] += 3
  puts counts["a"]
  puts counts["b"]
  puts counts["c"]
end

gauntlet("Hash.new default with methods") do
  counts = Hash.new(0)
  counts["x"] += 10
  counts["y"] += 20
  puts counts.keys.length
  puts counts.size
end

gauntlet("Hash.new default each") do
  counts = Hash.new(0)
  counts["a"] += 5
  counts["b"] += 3
  total = 0
  counts.each do |k, v|
    total += v
  end
  puts total
end

gauntlet("Hash.new with block (string default)") do
  h = Hash.new { |h, k| h[k] = "default_#{k}" }
  puts h["x"]
  puts h["y"]
  puts h.has_key?("x")
end

gauntlet("Hash.new with block (array accumulation)") do
  h = Hash.new { |h, k| h[k] = [] }
  h["fruits"] << "apple"
  h["fruits"] << "banana"
  h["vegs"] << "carrot"
  h["fruits"].each { |f| puts f }
  h["vegs"].each { |v| puts v }
end

gauntlet("Hash << on array values") do
  h = {"fruits" => ["apple"], "vegs" => ["carrot"]}
  h["fruits"] << "banana"
  h["vegs"] << "broccoli"
  h["fruits"].each { |f| puts f }
  h["vegs"].each { |v| puts v }
end

gauntlet("Hash.new with block (push)") do
  h = Hash.new { |h, k| h[k] = [] }
  h["a"].push("x")
  h["a"].push("y")
  h["b"].push("z")
  h["a"].each { |v| puts v }
  puts h["b"].length
end

gauntlet("Hash#all?") do
  h = {a: 1, b: 2, c: 3}
  puts h.all? { |k, v| v > 0 }
  puts h.all? { |k, v| v > 1 }
end

gauntlet("Hash#none?") do
  h = {a: 1, b: 2, c: 3}
  puts h.none? { |k, v| v > 10 }
  puts h.none? { |k, v| v > 2 }
end

gauntlet("Hash#flat_map") do
  h = {a: 1, b: 2, c: 3}
  result = h.flat_map { |k, v| [k.to_s, v.to_s] }
  result.each { |s| puts s }
end

gauntlet("Hash#sum") do
  h = {a: 1, b: 2, c: 3}
  puts h.sum { |k, v| v }
  puts h.sum { |k, v| v * 2 }
end

gauntlet("Hash#count with block") do
  h = {a: 1, b: 2, c: 3, d: 4}
  puts h.count { |k, v| v > 2 }
end

gauntlet("Hash#reduce") do
  h = {a: 1, b: 2, c: 3}
  total = h.reduce(0) do |acc, (k, v)|
    acc + v
  end
  puts total
end

gauntlet("Hash#inject") do
  h = {a: 10, b: 20, c: 30}
  result = h.inject("") do |acc, (k, v)|
    acc + k.to_s
  end
  puts result
end

gauntlet("Hash#each_with_index") do
  h = {a: 1, b: 2, c: 3}
  h.each_with_index do |(k, v), i|
    puts "#{i}: #{k}=#{v}"
  end
end

gauntlet("Hash#transform_values! same type") do
  h = {"a" => 1, "b" => 2, "c" => 3}
  h.transform_values! { |v| v * 10 }
  h.each { |k, v| puts "#{k}: #{v}" }
end

gauntlet("Hash#transform_values! type change") do
  h = {"a" => 1, "b" => 2, "c" => 3}
  h.transform_values! { |v| v.to_s }
  h.each { |k, v| puts "#{k}: #{v}" }
end

gauntlet("Hash#key") do
  h = {a: 1, b: 2, c: 3}
  puts h.key(2)
  puts h.key(3)
end

gauntlet("Hash#merge with block") do
  h1 = {a: 1, b: 2}
  h2 = {b: 3, c: 4}
  merged = h1.merge(h2) { |key, old_val, new_val| old_val + new_val }
  merged.keys.each { |k| puts "#{k}: #{merged[k]}" }
end

gauntlet("Hash#transform_keys") do
  h = {a: 1, b: 2, c: 3}
  result = h.transform_keys { |k| k.to_s.upcase }
  result.keys.each { |k| puts "#{k}: #{result[k]}" }
end

gauntlet("Hash#to_a") do
  h = {a: 1, b: 2, c: 3}
  pairs = h.to_a
  puts pairs.length
end

gauntlet("Hash#sort_by") do
  h = {c: 3, a: 1, b: 2}
  sorted = h.sort_by { |k, v| v }
  sorted.each { |k, v| puts "#{k}: #{v}" }
end

gauntlet("Hash#min_by") do
  h = {a: 3, b: 1, c: 2}
  k, v = h.min_by { |k, v| v }
  puts "#{k}: #{v}"
end

gauntlet("Hash#max_by") do
  h = {a: 3, b: 1, c: 2}
  k, v = h.max_by { |k, v| v }
  puts "#{k}: #{v}"
end

gauntlet("Hash#values_at") do
  h = {a: 1, b: 2, c: 3, d: 4}
  result = h.values_at(:b, :d)
  result.each { |v| puts v }
end

gauntlet("Hash#merge!") do
  h1 = {a: 1, b: 2}
  h2 = {b: 3, c: 4}
  h1.merge!(h2)
  h1.each { |k, v| puts "#{k}: #{v}" }
end

gauntlet("Hash#select!") do
  h = {a: 1, b: 2, c: 3, d: 4}
  h.select! { |k, v| v > 2 }
  h.each { |k, v| puts "#{k}: #{v}" }
end

gauntlet("Hash#reject!") do
  h = {a: 1, b: 2, c: 3, d: 4}
  h.reject! { |k, v| v > 2 }
  h.each { |k, v| puts "#{k}: #{v}" }
end

gauntlet("Hash#shift") do
  h = {a: 1, b: 2, c: 3}
  k, v = h.shift
  puts "#{k}: #{v}"
  puts h.length
end

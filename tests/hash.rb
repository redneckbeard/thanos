gauntlet("Hash#delete") do
  puts({:foo => "x", :bar => "y"}.delete(:foo))
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
  h.each do |k, v|
    puts "#{k}: #{v}"
  end
  puts smaller.length == h.length
end

gauntlet("Hash#values") do
  h = {foo: 1, bar: 2, baz: 3, quux: 4}
  if h.has_value?(3)
    h.values.sort!.each do |v|
      puts v
    end
  end
end

gauntlet("Hash#keys") do
  h = {foo: 1, bar: 2, baz: 3, quux: 4}
  if h.has_key?(:foo)
    h.keys.sort!.each do |k|
      puts k
    end
  end
end

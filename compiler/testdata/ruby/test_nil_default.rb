def greet(name = nil)
  name ||= "world"
  puts "hello #{name}"
end

greet("paul")
greet

def find_index(arr, target)
  arr.each_with_index do |val, i|
    return i if val == target
  end
  nil
end

puts find_index([10, 20, 30], 20)
puts find_index([10, 20, 30], 99).nil?

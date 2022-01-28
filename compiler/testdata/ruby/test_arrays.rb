def make_arr(a, b, c)
  arr = [a, b, c]
  arr << a * b * c
  if a > 10
    return [b]
  end
  arr
end

def sum(a)
  a.reduce(0) do |acc, n|
    acc + n
  end
end

def squares_plus_one(a)
  a.map do |i|
    squared = i*i
    squared + 1
  end
end

def double_third(a)
  a[2] * 2
end

def length_is_size(a)
  a.size == a.length
end

def swap_positions(a, b)
  return b, a
end

arr = make_arr(1, 2, 3)
qpo = squares_plus_one([1,2,3,4]).select do |x|
  x % 2 == 0
end.length
total = sum([1,2,3,4])
doubled = double_third([1,2,3])
foo = length_is_size([1,2,3])
i, b = swap_positions true, 10


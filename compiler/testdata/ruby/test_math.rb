x = 10 / 2
y = x / 2.0
z = x ** 2
a = x ** x
b = y ** 2
c = 12.0 / 4
d = -50.abs
e = x.abs

10.times do |x|
  if x.even?
    puts x
  end
end

15.downto(10) do |x|
  if x.odd?
    puts x
  end
end

-5.upto(5) do |x|
  case
  when x.zero?
    puts "zero"
  when x.positive?
    puts "positive"
  when x.negative?
    puts "negative"
  end
end

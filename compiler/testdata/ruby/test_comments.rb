# Top-level comment before code
x = 42

# Comment before conditional
if x > 10
  puts "big"
else
  puts "small"
end

# Comment before loop
arr = [1, 2, 3]
arr.each do |n|
  puts n
end

# Comment before assignment
y = x + 1
# Comment before last statement
puts y

# Multiple consecutive comments
# describe what the next
# block of code does
z = y * 2
puts z

=begin
Block comments are also preserved
They span multiple lines
=end
a = z + 1
puts a

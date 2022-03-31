x = 100

while x > 0 do
  x -= 1
end

until x == 50 do
  x += 1
end

y = 0

while x do
  y += 1
  if y > 5
    break
  end
end

until y > 100
  y += 1
  if y % 2 == 0
    next
  end
  puts y
end

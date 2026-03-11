gauntlet("Time.new with args") do
  t = Time.new(2024, 3, 15, 10, 30, 45)
  puts t.year
  puts t.month
  puts t.day
  puts t.hour
  puts t.min
  puts t.sec
end

gauntlet("Time#strftime") do
  t = Time.new(2024, 12, 25, 14, 30, 0)
  puts t.strftime("%Y-%m-%d")
  puts t.strftime("%H:%M:%S")
  puts t.strftime("%Y/%m/%d %H:%M")
end

gauntlet("Time#wday and yday") do
  t = Time.new(2024, 1, 1)
  puts t.wday
  puts t.yday
end

gauntlet("Time#to_i") do
  t = Time.new(2024, 1, 1, 0, 0, 0)
  puts t.to_i > 0
end

gauntlet("Time comparison") do
  a = Time.new(2024, 1, 1)
  b = Time.new(2024, 12, 31)
  puts a < b
  puts a > b
  puts a == a
end

gauntlet("Time#utc") do
  t = Time.new(2024, 6, 15, 12, 0, 0)
  u = t.utc
  puts u.hour
end

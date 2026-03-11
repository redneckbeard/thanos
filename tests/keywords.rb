gauntlet("Go keyword as variable name") do
  type = "hello"
  range = 42
  func = true
  select = "world"
  puts type
  puts range
  puts func
  puts select
end

gauntlet("Go keyword as method parameter") do
  def check(type, range)
    puts type
    puts range
  end
  check("foo", 10)
end

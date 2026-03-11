gauntlet("break") do
  10.times do |i|
    puts i
    if i > 5
      break
    end
  end
end

gauntlet("next") do
  10.times do |i|
    puts i
    if i % 2 == 0
      next
    end
  end
end

gauntlet("next postfix in each") do
  [1, 2, 3, 4, 5].each do |i|
    next if i == 3
    puts i
  end
end

gauntlet("break postfix in each") do
  [1, 2, 3, 4, 5].each do |i|
    break if i == 4
    puts i
  end
end

gauntlet("next in while") do
  i = 0
  while i < 10
    i += 1
    next if i % 2 == 0
    puts i
  end
end

gauntlet("next with value in map") do
  result = [1, 2, 3, 4, 5].map do |i|
    next(0) if i == 3
    i * 10
  end
  puts result.join(", ")
end

gauntlet("next with value negative") do
  result = [1, 2, 3, 4, 5].map do |x|
    next(-1) if x < 3
    x * 100
  end
  puts result.join(", ")
end

gauntlet("while loop") do
  x = 0
  while x < 5 do
    puts x
    x += 1
  end
end

gauntlet("until loop") do
  x = 0
  until x >= 5
    puts x
    x += 1
  end
end

gauntlet("for-in loop") do
  for x in [10, 20, 30, 40] do
    puts x
  end
end

gauntlet("if/elsif/else") do
  x = 15
  if x > 20
    puts "big"
  elsif x > 10
    puts "medium"
  else
    puts "small"
  end
end

gauntlet("case/when") do
  x = 3
  case x
  when 1
    puts "one"
  when 2
    puts "two"
  when 3
    puts "three"
  else
    puts "other"
  end
end

gauntlet("ternary-style if") do
  x = 10
  result = if x > 5 then "big" else "small" end
  puts result
end

gauntlet("ternary operator") do
  x = 10
  puts x > 5 ? "big" : "small"
  puts x < 5 ? "yes" : "no"
end

gauntlet("begin/rescue basic") do
  begin
    puts "try"
  rescue => e
    puts "caught"
  end
end

gauntlet("begin/rescue/ensure") do
  begin
    puts "body"
  rescue => e
    puts "rescue"
  ensure
    puts "ensure"
  end
end

gauntlet("begin/rescue without variable") do
  begin
    puts "hello"
  rescue
    puts "caught"
  end
end

gauntlet("raise with string") do
  begin
    raise "something went wrong"
  rescue => e
    puts "caught"
  end
end

gauntlet("raise with class") do
  begin
    raise ArgumentError, "bad arg"
  rescue ArgumentError
    puts "caught argument error"
  end
end

gauntlet("rescue specific class with variable") do
  begin
    raise TypeError, "wrong type"
  rescue TypeError => e
    puts e.message
  end
end

gauntlet("rescue multiple clauses") do
  begin
    raise ArgumentError, "bad"
  rescue TypeError
    puts "type error"
  rescue ArgumentError
    puts "argument error"
  end
end

gauntlet("rescue with fallthrough to catch-all") do
  begin
    raise "generic"
  rescue ArgumentError
    puts "arg error"
  rescue => e
    puts "caught other"
  end
end

gauntlet("rescue does not catch wrong type") do
  begin
    begin
      raise TypeError, "wrong"
    rescue ArgumentError
      puts "should not print"
    end
  rescue => e
    puts "outer caught"
  end
end

gauntlet("raise and ensure") do
  begin
    raise "boom"
  rescue => e
    puts "rescued"
  ensure
    puts "cleanup"
  end
end

gauntlet("lambda basic") do
  double = ->(x) { x * 2 }
  puts double.call(5)
  puts double.call(21)
end

gauntlet("lambda no args") do
  greeting = ->() { "hello world" }
  puts greeting.call
end

gauntlet("lambda multi params") do
  add = ->(a, b) { a + b }
  puts add.call(3, 4)
  puts add.call(10, 20)
end

gauntlet("lambda with strings") do
  greet = ->(name) { "hello " + name }
  puts greet.call("world")
  puts greet.call("Ruby")
end

gauntlet("lambda do/end form") do
  square = ->(x) do
    x * x
  end
  puts square.call(5)
  puts square.call(7)
end

gauntlet("case/when as expression") do
  n = 2
  result = case n
  when 1 then "one"
  when 2 then "two"
  else "other"
  end
  puts result
end

gauntlet("case/when as return value") do
  def classify(x)
    case x
    when 1 then "small"
    when 2 then "medium"
    else "large"
    end
  end
  puts classify(1)
  puts classify(2)
  puts classify(99)
end

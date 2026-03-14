gauntlet("defined? keyword") do
  if defined?(String)
    puts "yes"
  else
    puts "no"
  end
end

gauntlet("fail with postfix if in method") do
  def safe_divide(a, b)
    fail "division by zero" if b == 0
    a / b
  end

  puts safe_divide(10, 2)
  puts safe_divide(9, 3)
end

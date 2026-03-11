gauntlet("no-arg lambda without parens") do
  counter = 0
  inc = -> { counter += 1 }
  inc.call
  inc.call
  inc.call
  puts counter
end

gauntlet("range variable each") do
  result = []
  r = 0..3
  r.each { result << _1 }
  puts result.join(", ")
end

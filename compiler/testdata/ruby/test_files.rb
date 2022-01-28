f = File.new("stuff.txt")

f.each do |ln|
  puts ln.gsub(/good/, "bad")
end

f.close

puts File.open("writable.txt", "a+") do |f|
  f << "here are some bits"
  f.size # only here to prove that we get the return type of the block as the return type of the whole expression
end.is_a?(Integer)

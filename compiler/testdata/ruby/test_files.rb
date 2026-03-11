f = File.new("stuff.txt")

f.each do |ln|
  puts ln.gsub(/good/, "bad")
end

f.close

puts(File.open("writable.txt", "a+") do |f|
  f << "here are some bits"
  f.size # only here to prove that we get the return type of the block as the return type of the whole expression
end.is_a?(Integer))

File.open("readme.txt") do |f|
  f.each_line do |line|
    puts line
  end
end

puts File.basename("/usr/local/bin/ruby")
puts File.dirname("/usr/local/bin/ruby")
puts File.extname("test.rb")
puts File.exist?("stuff.txt")
puts File.directory?("stuff")

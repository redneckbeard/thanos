gauntlet("shelling out with backticks") do
  %w{date time awk sed}.each do |cmd|
    puts `man -P cat #{cmd}`
  end
end

gauntlet("escape sequences") do
  ["foo\n", "f\oo", 'f\oo', '\'', "\\\"" ].each do |s|
    puts s
  end
end

gauntlet("String#hex") do
  %w{0x0a -1234 0 wombat}.each do |s|
    puts s.hex
  end
end

gauntlet("String#split") do
  " now's  the time".split.each {|s| puts s}
  " now's  the time".split(' ').each {|s| puts s}
  " now's  the time".split(/ /).each {|s| puts s}
  "1, 2.34,56, 7".split(/,\s*/).each {|s| puts s}
  "hello".split(//).each {|s| puts s}
  "hello".split(//, 3).each {|s| puts s}
  "hi mom".split(/\s*/).each {|s| puts s}
  "mellow yellow".split("ello").each {|s| puts s}
  "1,2,,3,4,,".split(',').each {|s| puts s}
  "1,2,,3,4,,".split(',', 4).each {|s| puts s}
  "1,2,,3,4,,".split(',', -4).each {|s| puts s}
  #"1:2:3".split(/(:)()()/, 2).each {|s| puts s}
  "".split(',', -1).each {|s| puts s}
end

gauntlet("String#strip etc.") do
  puts "    hello    ".strip
  puts "\tgoodbye\r\n".strip
  puts "  hello  ".lstrip
  puts "  hello  ".rstrip
end

gauntlet("string indexes") do
  puts "foobarbarbaz"[0]
  puts "foobarbaz"[1..3]
end

gauntlet("String#reverse") do
  puts "hello".reverse
end

gauntlet("String#upcase and downcase") do
  puts "hello".upcase
  puts "HELLO".downcase
  puts "Hello World".capitalize
end

gauntlet("String#include?") do
  puts "hello world".include?("world")
  puts "hello world".include?("xyz")
end

gauntlet("String#start_with? and end_with?") do
  puts "hello world".start_with?("hello")
  puts "hello world".start_with?("world")
  puts "hello world".end_with?("world")
  puts "hello world".end_with?("hello")
end

gauntlet("String#size") do
  puts "hello".size
  puts "".size
end

gauntlet("String#empty?") do
  puts "hello".empty?
  puts "".empty?
end

gauntlet("String#to_i and to_f") do
  puts "42".to_i
  puts "3.14".to_f
end

gauntlet("String#*") do
  puts "ha" * 3
end

gauntlet("String#+ concatenation") do
  puts "hello" + " " + "world"
end

gauntlet("String#capitalize") do
  puts "hello world".capitalize
  puts "HELLO".capitalize
end

gauntlet("string interpolation") do
  name = "world"
  age = 42
  puts "hello #{name}, age #{age}"
end

gauntlet("String#chars") do
  "hello".chars.each { |c| puts c }
end

gauntlet("String#chomp") do
  puts "hello\n".chomp
  puts "hello".chomp
end

gauntlet("String#gsub") do
  puts "hello world hello".gsub(/hello/, "goodbye")
end

gauntlet("String#sub") do
  puts "hello world hello".sub("hello", "goodbye")
end

gauntlet("String#sub with regex") do
  puts "hello world hello".sub(/h\w+/, "goodbye")
  puts "2026-03-09".sub(/(\d{4})-(\d{2})-(\d{2})/, '\2/\3/\1')
end

gauntlet("String#count") do
  puts "hello world".count("l")
end

gauntlet("String#split") do
  "hello,world,foo".split(",").each { |s| puts s }
end

gauntlet("String#lstrip and rstrip") do
  puts "  hello  ".lstrip
  puts "  hello  ".rstrip
  puts "  hello  ".strip
end

gauntlet("String#scan") do
  "hello world hello".scan(/hello/).each { |m| puts m }
  "one 1 two 2 three 3".scan(/\d+/).each { |m| puts m }
end

gauntlet("String#center") do
  puts "hi".center(10)
  puts "hi".center(10, "-")
  puts "hello world".center(5)
end

gauntlet("String#ljust") do
  puts "hi".ljust(10)
  puts "hi".ljust(10, "*")
  puts "hello world".ljust(5)
end

gauntlet("String#rjust") do
  puts "hi".rjust(10)
  puts "hi".rjust(10, "0")
  puts "hello world".rjust(5)
end

gauntlet("String#tr") do
  puts "hello".tr("el", "ip")
  puts "hello".tr("aeiou", "*")
end

gauntlet("String comparisons") do
  puts "abc" < "def"
  puts "def" > "abc"
  puts "abc" == "abc"
  puts "abc" != "def"
end

gauntlet("String#gsub with string args") do
  puts "hello world hello".gsub("hello", "goodbye")
  puts "aaa".gsub("a", "bb")
end

gauntlet("String#length") do
  puts "hello".length
  puts "".length
end

gauntlet("String#% single arg") do
  puts "hello %s" % "world"
end

gauntlet("String#% multiple args") do
  puts "x=%d y=%d" % [10, 20]
end

gauntlet("String#% with variable") do
  name = "Alice"
  puts "hello %s!" % name
end

gauntlet("String#% mixed types") do
  name = "Alice"
  age = 30
  puts "%s is %d years old" % [name, age]
end

gauntlet("String#freeze") do
  s = "hello".freeze
  puts s
end

gauntlet("String#match?") do
  puts "hello world".match?(/world/)
  puts "hello world".match?(/xyz/)
end

gauntlet("String#bytes") do
  "ABC".bytes.each { |b| puts b }
end

gauntlet("String#delete") do
  puts "hello world".delete("lo")
end

gauntlet("String#replace") do
  s = "hello"
  s.replace("world")
  puts s
end

gauntlet("negative string indexing") do
  s = "hello"
  puts s[-1]
  puts s[-2]
end

gauntlet("String#freeze") do
  s = "hello".freeze
  puts s
end

gauntlet("String#bytes") do
  "ABC".bytes.each { |b| puts b }
end

gauntlet("String#<<") do
  s = "hello"
  s << " world"
  puts s
end

gauntlet("String#delete_prefix") do
  puts "hello world".delete_prefix("hello ")
  puts "hello world".delete_prefix("xyz")
end

gauntlet("String#delete_suffix") do
  puts "hello world".delete_suffix(" world")
  puts "hello world".delete_suffix("xyz")
end

gauntlet("String#each_char") do
  "abc".each_char { |c| puts c }
end

gauntlet("String#each_line") do
  "hello\nworld\nfoo".each_line { |line| puts line }
end

gauntlet("String#squeeze") do
  puts "aaabbbccc".squeeze
  puts "aabbccdd".squeeze("b")
end

gauntlet("String#index") do
  puts "hello world".index("world")
  puts "hello world".index("o")
end

gauntlet("String#lines") do
  "one\ntwo\nthree".lines.each { |l| puts l }
end

gauntlet("String#prepend") do
  s = "world"
  s.prepend("hello ")
  puts s
end

gauntlet("String#gsub with block") do
  result = "hello world".gsub(/[aeiou]/) { |m| m.upcase }
  puts result
end

gauntlet("String#[] index") do
  puts "hello"[1]
  puts "hello"[0]
  puts "hello"[-1]
end

gauntlet("String#[] range") do
  puts "hello world"[0..4]
  puts "hello world"[6..10]
end

gauntlet("String#ord") do
  puts "A".ord
  puts "a".ord
end

gauntlet("String#swapcase") do
  puts "Hello World".swapcase
end

gauntlet("String#chop") do
  puts "hello".chop
  puts "hello\n".chop
end

gauntlet("String#casecmp?") do
  puts "hello".casecmp?("HELLO")
  puts "hello".casecmp?("world")
end

gauntlet("String#partition") do
  a, b, c = "hello-world-foo".partition("-")
  puts a
  puts b
  puts c
end

gauntlet("String#rpartition") do
  a, b, c = "hello-world-foo".rpartition("-")
  puts a
  puts b
  puts c
end

gauntlet("String#rindex") do
  puts "hello world hello".rindex("hello")
end

gauntlet("String#chr") do
  puts "hello".chr
end

gauntlet("String#between?") do
  puts "cat".between?("ant", "dog")
  puts "zebra".between?("ant", "dog")
end

gauntlet("String#codepoints") do
  "abc".codepoints.each { |c| puts c }
end

gauntlet("String#succ") do
  puts "a".succ
  puts "az".succ
  puts "zz".succ
  puts "9".succ
  puts "abc99".succ
end

gauntlet("String#oct") do
  puts "77".oct
  puts "10".oct
end

gauntlet("String#bytesize") do
  puts "hello".bytesize
end

gauntlet("String#clear") do
  s = "hello"
  s.clear
  puts s.length
end

gauntlet("String#insert") do
  puts "hello".insert(2, "XY")
end

gauntlet("String#concat") do
  s = "hello"
  s.concat(" world")
  puts s
end

gauntlet("String#upcase!") do
  s = "hello"
  s.upcase!
  puts s
end

gauntlet("String#downcase!") do
  s = "HELLO"
  s.downcase!
  puts s
end

gauntlet("String#strip!") do
  s = "  hello  "
  s.strip!
  puts s
end

gauntlet("String#lstrip!") do
  s = "  hello  "
  s.lstrip!
  puts s
end

gauntlet("String#rstrip!") do
  s = "  hello  "
  s.rstrip!
  puts s
end

gauntlet("String#chomp!") do
  s = "hello\n"
  s.chomp!
  puts s
end

gauntlet("String#chop!") do
  s = "hello"
  s.chop!
  puts s
end

gauntlet("String#sub!") do
  s = "hello world"
  s.sub!("world", "ruby")
  puts s
end

gauntlet("String#sub! with regex") do
  s = "hello world hello"
  s.sub!(/h\w+/, "goodbye")
  puts s
end

gauntlet("String#capitalize!") do
  s = "hello"
  s.capitalize!
  puts s
end

gauntlet("String#swapcase!") do
  s = "Hello World"
  s.swapcase!
  puts s
end

gauntlet("String#squeeze!") do
  s = "aaabbbccc"
  s.squeeze!
  puts s
end

gauntlet("String#encode") do
  puts "hello".encode("UTF-8")
end

gauntlet("String#gsub!") do
  s = "hello world hello"
  s.gsub!("hello", "bye")
  puts s
end

gauntlet("String#delete!") do
  s = "hello"
  s.delete!("l")
  puts s
end

gauntlet("String#delete_prefix!") do
  s = "hello world"
  s.delete_prefix!("hello ")
  puts s
end

gauntlet("String#delete_suffix!") do
  s = "hello world"
  s.delete_suffix!(" world")
  puts s
end

gauntlet("String#upto") do
  "a".upto("e") { |s| puts s }
end

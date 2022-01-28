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

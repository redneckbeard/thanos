gauntlet("match") do
  ["football", "goosefoot", "tomfoolery"].each do |cand|
    puts cand.match(/foo(?<tail>.+)/)["tail"]
  end
end

gauntlet("gsub") do
  ["football", "goosefoot", "tomfoolery"].each do |cand|
    puts cand.gsub(/foo(?<tail>.+)/, 'bar\k<tail>')
  end
end

gauntlet("=~ operator as boolean") do
  if "hello world" =~ /world/
    puts "matched"
  end
  if "hello world" =~ /xyz/
    puts "should not print"
  end
end

gauntlet("regex case-insensitive flag") do
  if "Hello" =~ /hello/i
    puts "matched"
  end
  puts "HELLO WORLD".gsub(/hello/i, "goodbye")
  puts "ABC".scan(/[a-z]+/i).length
end

gauntlet("regex multiline flag") do
  text = "first\nsecond"
  if text =~ /first.second/m
    puts "matched"
  end
end

gauntlet("gsub with hash arg") do
  replacements = {"a" => "*", "e" => "#", "i" => "!", "o" => "@", "u" => "$"}
  puts "hello world".gsub(/[aeiou]/, replacements)
end

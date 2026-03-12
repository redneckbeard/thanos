gauntlet("Shellwords.escape simple") do
  require 'shellwords'
  puts Shellwords.escape("hello")
end

gauntlet("Shellwords.escape with spaces") do
  require 'shellwords'
  puts Shellwords.escape("hello world")
end

gauntlet("Shellwords.escape special chars") do
  require 'shellwords'
  puts Shellwords.escape("it's")
end

gauntlet("Shellwords.split simple") do
  require 'shellwords'
  words = Shellwords.split("hello world")
  words.each { |w| puts w }
end

gauntlet("Shellwords.split quoted") do
  require 'shellwords'
  words = Shellwords.split('one "two three" four')
  words.each { |w| puts w }
end

gauntlet("Shellwords.split single quoted") do
  require 'shellwords'
  words = Shellwords.split("one 'two three' four")
  words.each { |w| puts w }
end

gauntlet("Shellwords.join") do
  require 'shellwords'
  puts Shellwords.join(["hello", "world"])
end

gauntlet("Shellwords.join with spaces") do
  require 'shellwords'
  puts Shellwords.join(["hello", "big world", "test"])
end

gauntlet("Shellwords.shellescape alias") do
  require 'shellwords'
  puts Shellwords.shellescape("hello world")
end

gauntlet("Shellwords.shellsplit alias") do
  require 'shellwords'
  words = Shellwords.shellsplit("a b c")
  words.each { |w| puts w }
end

gauntlet("Shellwords.shelljoin alias") do
  require 'shellwords'
  puts Shellwords.shelljoin(["a", "b c"])
end

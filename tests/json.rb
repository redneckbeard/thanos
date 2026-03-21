gauntlet("Hash#to_json") do
  require 'json'
  h = { name: "Alice", city: "NYC" }
  puts h.to_json
end

gauntlet("Array#to_json") do
  require 'json'
  arr = [1, 2, 3]
  puts arr.to_json
end

gauntlet("String#to_json") do
  require 'json'
  puts "hello".to_json
end

gauntlet("JSON.generate hash") do
  require 'json'
  h = { name: "Bob", age: "30" }
  puts JSON.generate(h)
end

gauntlet("nested Hash#to_json") do
  require 'json'
  h = { greeting: "hello", target: "world" }
  puts h.to_json
end

gauntlet("JSON.parse") do
  require 'json'
  h = JSON.parse("{\"name\":\"Alice\",\"age\":\"30\"}")
  puts h["name"]
  puts h["age"]
end

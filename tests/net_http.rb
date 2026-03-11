gauntlet("Net::HTTP.get") do
  require 'net/http'
  body = Net::HTTP.get("example.com", "/")
  puts body.length > 0
end

gauntlet("Net::HTTP.get_response") do
  require 'net/http'
  response = Net::HTTP.get_response("example.com", "/")
  puts response.code
  puts response.body.length > 0
end

gauntlet("Net::HTTP.get_response message") do
  require 'net/http'
  response = Net::HTTP.get_response("example.com", "/")
  puts response.message.length > 0
end

gauntlet("Net::HTTP.start with block") do
  require 'net/http'
  Net::HTTP.start("example.com", 80) do |http|
    response = http.get("/")
    puts response.code
    puts response.body.length > 0
  end
end

gauntlet("Net::HTTP.start without block") do
  require 'net/http'
  http = Net::HTTP.start("example.com", 80)
  response = http.get("/")
  puts response.code
end

gauntlet("Net::HTTP.new") do
  require 'net/http'
  http = Net::HTTP.new("example.com", 80)
  response = http.get("/")
  puts response.code
end

gauntlet("Net::HTTP.start with use_ssl") do
  require 'net/http'
  Net::HTTP.start("example.com", 443, use_ssl: true) do |http|
    response = http.get("/")
    puts response.code
  end
end

gauntlet("Net::HTTP response header") do
  require 'net/http'
  response = Net::HTTP.get_response("example.com", "/")
  puts response["Content-Type"].length > 0
end

gauntlet("Net::HTTP instance head") do
  require 'net/http'
  Net::HTTP.start("example.com", 80) do |http|
    response = http.head("/")
    puts response.code
  end
end

gauntlet("Net::HTTP::Get.new with request") do
  require 'net/http'
  Net::HTTP.start("example.com", 80) do |http|
    req = Net::HTTP::Get.new("/")
    req["Accept"] = "text/html"
    response = http.request(req)
    puts response.code
    puts response.body.length > 0
  end
end

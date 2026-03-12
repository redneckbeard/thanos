gauntlet("URI.parse scheme") do
  require 'uri'
  u = URI.parse("https://example.com/path")
  puts u.scheme
end

gauntlet("URI.parse host") do
  require 'uri'
  u = URI.parse("https://example.com:8080/path")
  puts u.host
end

gauntlet("URI.parse path") do
  require 'uri'
  u = URI.parse("https://example.com/foo/bar")
  puts u.path
end

gauntlet("URI.parse query") do
  require 'uri'
  u = URI.parse("https://example.com/path?key=val&a=b")
  puts u.query
end

gauntlet("URI.parse fragment") do
  require 'uri'
  u = URI.parse("https://example.com/path#section")
  puts u.fragment
end

gauntlet("URI.parse to_s") do
  require 'uri'
  u = URI.parse("https://example.com/path?q=1")
  puts u.to_s
end

gauntlet("URI.encode_www_form_component") do
  require 'uri'
  puts URI.encode_www_form_component("hello world")
  puts URI.encode_www_form_component("a&b=c")
end

gauntlet("URI.decode_www_form_component") do
  require 'uri'
  puts URI.decode_www_form_component("hello+world")
  puts URI.decode_www_form_component("a%26b%3Dc")
end

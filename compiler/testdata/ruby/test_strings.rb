def hello(name)
  puts "debug message"
  "Hello, " + name
end

def hello_interp(name, age)
  comparative = if age > 40
                  "older"
                else
                  "younger"
                end
  puts "#{name} is #{comparative} than me, age #{age}"
end

def matches_foo(foolike)
  if /foo/ =~ foolike
    puts "got a match"
  end
end

def matches_interp(foo, bar)
  if /foo#{foo}/ =~ bar
    puts "got a match"
  end
end

def extract_third_octet(ip)
  ip.match(/\d{1,3}\.\d{1,3}\.(?<third>\d{1,3})\.\d{1,3}/)["third"]
end

greeting = hello("me")
hello_interp("Steve", 38)
matches_foo("football")
matches_interp(10, "foofoo")
extract_third_octet("127.0.0.1")
terms = %w{foo bar baz}
interp_terms = %W{foo #{"BAR BAZ QUUX"} bar}
puts `man -P cat #{"date"}`

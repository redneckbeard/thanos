gauntlet("String#[offset, length]=") do
  s = "hello world"
  s[0, 1] = "H"
  puts s

  s = "hello"
  s[5, 0] = " world"
  puts s

  s = "abcdef"
  s[2, 3] = "X"
  puts s
end

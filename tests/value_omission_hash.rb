gauntlet("value omission hash") do
  action = "+"
  position = "3"
  element = "hello"
  h = {action:, position:, element:}
  puts h[:action]
  puts h[:position]
  puts h[:element]
end

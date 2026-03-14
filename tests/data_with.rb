gauntlet("Data#with") do
  Change = Data.define(:action, :position, :element)
  c = Change.new("+", 3, "hello")
  puts c.action
  puts c.position
  puts c.element

  c2 = c.with(action: "-")
  puts c2.action
  puts c2.position
  puts c2.element

  c3 = c.with(position: 10, element: "world")
  puts c3.action
  puts c3.position
  puts c3.element
end

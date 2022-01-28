gauntlet("methods") do
  meths = [].methods
  # comes from Object
  puts meths.include?(:methods)
  # comes from Array
  puts meths.include?(:join)
end

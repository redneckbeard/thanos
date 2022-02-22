gauntlet("methods") do
  # comes from Object
  puts [].methods.include?(:methods)
  # comes from Array
  puts [].methods.include?(:join)
end

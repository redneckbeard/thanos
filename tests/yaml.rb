gauntlet("YAML.dump string") do
  require 'yaml'
  puts YAML.dump("hello")
end

gauntlet("YAML.dump integer") do
  require 'yaml'
  puts YAML.dump(42)
end

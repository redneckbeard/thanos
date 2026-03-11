gauntlet("symbol to proc map") do
  puts [1, 2, 3].map(&:to_s).join(", ")
end

gauntlet("symbol to proc select") do
  puts [-2, -1, 0, 1, 2].select(&:positive?).length
end

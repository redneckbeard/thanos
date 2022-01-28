gauntlet("match") do
  candidates = ["football", "goosefoot", "tomfoolery"]
  candidates.each do |cand|
    puts cand.match(/foo(?<tail>.+)/)["tail"]
  end
end

gauntlet("gsub") do
  ["football", "goosefoot", "tomfoolery"].each do |cand|
    puts cand.gsub(/foo(?<tail>.+)/, 'bar\k<tail>')
  end
end

gauntlet("diff-lcs lcs basic") do
  require "diff-lcs"
  a = [1, 2, 3, 4, 5]
  b = [1, 3, 5, 7]
  result = Diff::LCS.lcs(a, b)
  puts result.length
  result.each { |x| puts x }
end

gauntlet("diff-lcs lcs strings") do
  require "diff-lcs"
  a = "abcdef".chars
  b = "abcxef".chars
  result = Diff::LCS.lcs(a, b)
  puts result.join("")
end

gauntlet("diff-lcs lcs identical") do
  require "diff-lcs"
  a = [1, 2, 3]
  b = [1, 2, 3]
  result = Diff::LCS.lcs(a, b)
  puts result.length
end

gauntlet("diff-lcs lcs disjoint") do
  require "diff-lcs"
  a = [1, 2, 3]
  b = [4, 5, 6]
  result = Diff::LCS.lcs(a, b)
  puts result.length
end

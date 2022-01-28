gauntlet("File#each") do
  File.new("compiler/testdata/input/millennials.txt").each do |line|
    puts line.gsub(/[mM]illennial(?<plural>s)?/, 'Snake Person\k<plural>')
  end
end

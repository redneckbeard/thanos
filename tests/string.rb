gauntlet("shelling out with backticks") do
  %w{date time awk sed}.each do |cmd|
    puts `man -P cat #{cmd}`
  end
end

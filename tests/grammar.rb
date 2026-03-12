gauntlet("fcall command_args brace_block") do
  def apply(x)
    yield x
  end
  result = apply 5 do |n| n * 2 end
  puts result
end

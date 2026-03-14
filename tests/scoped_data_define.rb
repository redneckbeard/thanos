gauntlet("scoped Data.define") do
  module Diff
  end
  module Diff::LCS
  end
  Diff::LCS::Change = Data.define(:action, :position, :element)
  c = Diff::LCS::Change.new("-", 0, "hello")
  puts c.action
  puts c.position
  puts c.element
end

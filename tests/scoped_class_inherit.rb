gauntlet("scoped class with inheritance") do
  module Diff
    module LCS
    end
  end

  class Diff::LCS::Base
    def base_method
      "from base"
    end
  end

  class Diff::LCS::Child < Diff::LCS::Base
    def child_method
      base_method + " and child"
    end
  end

  c = Diff::LCS::Child.new
  puts c.child_method
end

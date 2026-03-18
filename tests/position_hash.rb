gauntlet("position hash") do
  module Internals
    def self.position_hash(enum, interval)
      hash = Hash.new { |h, k| h[k] = [] }
      interval.each { hash[enum[_1]] << _1 }
      hash
    end
  end

  b = [3, 1, 4, 1, 5, 9]
  h = Internals.position_hash(b, 0..5)
  puts h.size
end

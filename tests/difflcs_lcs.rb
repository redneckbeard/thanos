gauntlet("diff-lcs basic lcs") do
  module Diff
    module LCS
    end
  end

  # Scoped Data.define
  Diff::LCS::Change = Data.define(:action, :position, :element)

  # Simplified LCS algorithm
  class Diff::LCS::Algorithm
    def self.lcs(a, b)
      m = a.length
      n = b.length

      # Build the LCS table
      table = []
      i = 0
      while i <= m
        row = []
        j = 0
        while j <= n
          row.push(0)
          j = j + 1
        end
        table.push(row)
        i = i + 1
      end

      i = 1
      while i <= m
        j = 1
        while j <= n
          if a[i - 1] == b[j - 1]
            table[i][j] = table[i - 1][j - 1] + 1
          elsif table[i - 1][j] >= table[i][j - 1]
            table[i][j] = table[i - 1][j]
          else
            table[i][j] = table[i][j - 1]
          end
          j = j + 1
        end
        i = i + 1
      end

      # Backtrack to find the LCS
      result = []
      i = m
      j = n
      while i > 0 && j > 0
        if a[i - 1] == b[j - 1]
          result.push(a[i - 1])
          i = i - 1
          j = j - 1
        elsif table[i - 1][j] >= table[i][j - 1]
          i = i - 1
        else
          j = j - 1
        end
      end

      result.reverse
    end
  end

  # Test basic LCS
  a = %w[a b c d e f]
  b = %w[a c d f g]
  lcs = Diff::LCS::Algorithm.lcs(a, b)
  puts lcs.join(" ")
  puts lcs.length

  # Test scoped Data.define
  c = Diff::LCS::Change.new("+", 2, "d")
  puts c.action
  puts c.position
  puts c.element
end

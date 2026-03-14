gauntlet("nested array while top level") do
  a = ["a", "b", "c"]
  b = ["a", "c"]
  m = a.length
  n = b.length

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
  j = 0
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

  puts table[m][n]
end

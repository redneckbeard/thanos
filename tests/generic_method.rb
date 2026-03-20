gauntlet("generic method with int and string arrays") do
  def count_common(a, b)
    count = 0
    a.each do |x|
      b.each do |y|
        if x == y
          count = count + 1
        end
      end
    end
    count
  end

  puts count_common([1, 2, 3, 4], [3, 4, 5, 6])
  puts count_common(["a", "b", "c"], ["b", "c", "d"])

  def find_length(arr)
    arr.length
  end

  puts find_length([10, 20, 30])
  puts find_length(["x", "y"])
end

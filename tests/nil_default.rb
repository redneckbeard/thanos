gauntlet("nil default without arg") do
  def greet(name = nil)
    name ||= "world"
    name
  end

  puts greet
end

gauntlet("nil default with arg") do
  def greet(name = nil)
    name ||= "world"
    name
  end

  puts greet("paul")
end

gauntlet("optional return found") do
  def find_index(arr, target)
    arr.each_with_index do |val, i|
      return i if val == target
    end
    nil
  end

  puts find_index([10, 20, 30], 20)
end

gauntlet("optional return nil") do
  def find_index(arr, target)
    arr.each_with_index do |val, i|
      return i if val == target
    end
    nil
  end

  puts find_index([10, 20, 30], 99).nil?
end

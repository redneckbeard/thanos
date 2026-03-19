gauntlet("nil-init var reassigned to int") do
  def find_big(values)
    k = nil
    values.each do |v|
      if v > 15
        k = v
      end
    end
    if k.nil?
      puts "nil"
    else
      puts k
    end
  end

  find_big([10, 20, 30])
  find_big([1, 2, 3])
end

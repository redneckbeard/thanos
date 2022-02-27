gauntlet "user-defined block method" do
  def logging(arr, &blk)
    arr.each do |n|
      blk.call(n)
    end
  end

  logging([1,2,3,4,5]) do |x|
    puts x
  end
end

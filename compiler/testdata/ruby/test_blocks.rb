def foo(x, y, &blk)
  x * blk.call(y)
end

foo(7, 8) do |b|
  b / 10.0
end

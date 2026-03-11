def describe(n)
  result = case n
  when 1 then "one"
  when 2 then "two"
  else "other"
  end
  puts result
end

def categorize(n)
  case n
  when 1 then "small"
  when 2 then "medium"
  else "large"
  end
end

describe(1)
categorize(1)

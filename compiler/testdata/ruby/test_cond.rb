def cond_return(a, b)
  return a * b if a == 47
  if a < 0 && b < 0
    0
  elsif a >= b
    a
  else
    b
  end
end

def cond_assignment(a, b, c)
  foo = if a == b
    true
  else
    false
  end
  foo || c
end

def cond_invoke
  if true
    puts "it's true"
  else
    puts "it's false"
  end
  10
end

def tern(x, y, z)
  return 99 unless z < 50
  x == 10 ? y : z
end

def length_if_array(arr)
  # since we infer the type signature of the method at compile time, the following condition becomes essentially tautological and can just be compiled away entirely
  if arr.is_a?(Object)
     arr.size
  else
     0
  end
end

def puts_if_not_symbol
  if "foo".is_a?(Symbol)
    puts "is a symbol"
  else
    puts "isn't a symbol"
  end
end

def switch_on_int_val(x)
  case x
  when 0
    "none"
  when 1
    "one"
  when 2, 3, 4, 5
    "a few"
  else
    "many"
  end
end

def switch_on_int_with_range(x)
  case x
  when 0
    "none"
  when 1
    "one"
  when 2..5
    "a few"
  when 6, 7, 8 # In Go, this now has to get expanded to several expressions with ||
    "several"
  else
    "many"
  end
end

baz = cond_return(2, 4)
quux = cond_assignment(1, 3, false)
zoo = cond_invoke
last = tern(10, 20, 30)
length_if_array(["foo", "bar", "baz"])
switch_on_int_val(5)
switch_on_int_with_range(5)

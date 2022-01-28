class Foo
  def a
    true
  end

  def b(x, y)
    @baz = x + y
  end
end

class Bar < Foo
  def a
    super
  end

  def b(x, y)
    super
  end
end

Bar.new.b(1.5, 2)

class Baz < Foo
  def b(x, y)
    super(1.4, 2) + 1.0
  end
end

Baz.new.b(1.5, 2)

class Quux < Foo
  def a
    anc = super
    !anc
  end

  def b(x, y)
    anc = super(x**2, y**2)
    anc + 1
  end
end

quux = Quux.new
if quux.a
  puts quux.b(2.0, 4)
end

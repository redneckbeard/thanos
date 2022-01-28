gauntlet("simple class, no attrs") do
  class Foo
    def swap(dot_separated)
      dot_separated.gsub(/(\w+)\.(\w+)/, '\2.\1')
    end
  end
  puts Foo.new.swap("left.right")
end

gauntlet("simple class, methods reference other methods") do
  class Foo
    def swap(dot_separated)
      dot_separated.gsub(patt(), '\2.\1')
    end

    def patt
      /(\w+)\.(\w+)/
    end
  end
  puts Foo.new.swap("left.right")
end

gauntlet("simple class, methods reference methods in outer scope") do
  def unit(oz)
    if oz > 16 then "lbs" else "oz" end
  end

  class Foo
    def format(oz)
      "#{to_lbs(oz)} #{unit(oz)}"
    end

    def to_lbs(oz)
      if oz > 16 then oz / 16 else oz end
    end
  end
  puts Foo.new.format(18)
end

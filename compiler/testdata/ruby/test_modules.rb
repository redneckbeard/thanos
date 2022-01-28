Quux = false

module Foo
  Quux = "quux"

  class Baz
    Quux = 10

    def quux
      Quux
    end
  end
end

puts Foo::Baz.new.quux

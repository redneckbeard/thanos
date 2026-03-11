# Known Issues

## Module method returning module-scoped class instance

A `def self.x` method inside a module that returns an instance of a class defined in the same module generates the wrong Go type name. The class gets a qualified Go name (e.g., `AnimalsDog`) but the method's return type resolves to the unqualified name (`Dog`).

```ruby
module Animals
  class Dog
    def initialize(name)
      @name = name
    end
    def speak
      "#{@name} says woof"
    end
  end

  def self.create_dog(name)
    Dog.new(name)  # return type resolves to *Dog, should be *AnimalsDog
  end
end

dog = Animals.create_dog("Rex")
puts dog.speak
```

**Root cause**: `GoType()` on the class Instance returns the unqualified name when referenced from within the module scope. The compiler qualifies it during `CompileClass` but the type system doesn't propagate the qualified name back.

## Module methods cannot call other module methods

When a `def self.x` method inside a module calls another `def self.y` method on the same module without an explicit receiver, the call isn't routed to the module's own method specs. The param types on the callee don't get inferred.

```ruby
module Converter
  def self.fahrenheit_to_celsius(f)
    (f - 32) * 5 / 9
  end

  def self.format_temp(f)
    c = fahrenheit_to_celsius(f)  # unresolved — no implicit self for modules
    "#{f}F = #{c}C"
  end
end
```

**Root cause**: Module class methods are stored in `Module.ClassMethods`, not in the module's `MethodSet`. When `fahrenheit_to_celsius(f)` is parsed without a receiver, `AddCall` routes it to the module's MethodSet (or global), where no matching method exists. Class methods on classes have the same limitation but it's less noticeable because class methods rarely call each other without `self.`.

**Workaround**: Use explicit receiver: `Converter.fahrenheit_to_celsius(f)`.

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

gauntlet("class with initialize and attr_reader") do
  class Dog
    attr_reader :name, :age

    def initialize(name, age)
      @name = name
      @age = age
    end

    def speak
      "#{@name} says woof"
    end
  end

  d = Dog.new("Rex", 5)
  puts d.speak
  puts d.name
  puts d.age
end

gauntlet("class with inheritance") do
  class Animal
    def initialize(name)
      @name = name
    end

    def greet
      "I am #{@name}"
    end
  end

  class Cat < Animal
  end

  puts Cat.new("Whiskers").greet
end

gauntlet("constants") do
  PI = 3
  puts PI
end

gauntlet("module constants") do
  module Config
    VERSION = 42
  end

  puts Config::VERSION
end

gauntlet("inherited method with ivar in interpolation called on child") do
  class Animal
    def initialize(name)
      @name = name
    end

    def info
      "I am #{@name}"
    end
  end

  class Dog < Animal
  end

  puts Dog.new("Rex").info
end

gauntlet("child class with super and own initialize") do
  class Vehicle
    def initialize(make)
      @make = make
    end

    def info
      "Vehicle: #{@make}"
    end
  end

  class Car < Vehicle
    def initialize(make, model)
      super(make)
      @model = model
    end

    def full_info
      "#{info} #{@model}"
    end
  end

  puts Car.new("Toyota", "Camry").info
  puts Car.new("Honda", "Civic").full_info
end

gauntlet("inherited method called from child") do
  class Animal
    def greet
      "Hello from Animal"
    end
  end

  class Cat < Animal
    def speak
      greet
    end
  end

  puts Cat.new.speak
end

gauntlet("alias method") do
  class Greeter
    def hello(name)
      "Hello, #{name}!"
    end
    alias greet hello
  end
  g = Greeter.new
  puts g.hello("world")
  puts g.greet("Ruby")
end

gauntlet("class variable @@count") do
  class Counter
    @@count = 0

    def initialize
      @@count = @@count + 1
    end

    def self_count
      @@count
    end
  end

  Counter.new
  Counter.new
  Counter.new
  puts Counter.new.self_count
end

gauntlet("class variable shared across instances") do
  class Tracker
    @@total = 0

    def initialize(name)
      @name = name
      @@total = @@total + 1
    end

    def info
      "#{@name}: #{@@total}"
    end
  end

  Tracker.new("a")
  Tracker.new("b")
  puts Tracker.new("c").info
end

gauntlet("global variable") do
  $count = 0
  $count = $count + 1
  $count = $count + 1
  $count = $count + 1
  puts $count
end

gauntlet("global variable in method") do
  $total = 0
  def add_to_total(n)
    $total = $total + n
  end
  add_to_total(10)
  add_to_total(20)
  puts $total
end

gauntlet("open class adds method") do
  class Dog
    def initialize(name)
      @name = name
    end

    def speak
      "#{@name} says woof"
    end
  end

  class Dog
    def age
      5
    end
  end

  d = Dog.new("Rex")
  puts d.speak
  puts d.age
end

gauntlet("alias_method") do
  class Greeter
    def hello(name)
      "Hello, #{name}!"
    end
    alias_method :greet, :hello
  end
  g = Greeter.new
  puts g.hello("world")
  puts g.greet("Ruby")
end

gauntlet("include Comparable") do
  class Weight
    include Comparable
    attr_reader :value

    def initialize(value)
      @value = value
    end

    def <=>(other)
      @value - other.value
    end
  end

  a = Weight.new(10)
  b = Weight.new(20)
  c = Weight.new(10)

  puts a < b
  puts a > b
  puts a <= c
  puts a >= c
  puts a == c
  puts b > a
  puts a.between?(Weight.new(5), Weight.new(15))
  puts a.between?(Weight.new(15), Weight.new(25))
  puts b.clamp(Weight.new(5), Weight.new(15)).value
  puts a.clamp(Weight.new(5), Weight.new(15)).value
end

gauntlet("class method with yield") do
  class Items
    def initialize(items)
      @items = items
    end

    def each(&blk)
      @items.each do |x|
        blk.call(x)
      end
    end
  end

  list = Items.new([1, 2, 3])
  list.each { |x| puts x }
end

gauntlet("include Enumerable map") do
  class Numbers
    include Enumerable

    def initialize(items)
      @items = items
    end

    def each(&blk)
      @items.each do |x|
        blk.call(x)
      end
    end
  end

  nums = Numbers.new([1, 2, 3])
  doubled = nums.map { |x| x * 2 }
  doubled.each { |x| puts x }
end

gauntlet("include Enumerable select") do
  class Numbers
    include Enumerable

    def initialize(items)
      @items = items
    end

    def each(&blk)
      @items.each do |x|
        blk.call(x)
      end
    end
  end

  nums = Numbers.new([1, 2, 3, 4, 5])
  evens = nums.select { |x| x % 2 == 0 }
  evens.each { |x| puts x }
end

gauntlet("include Enumerable any? all? none?") do
  class Numbers
    include Enumerable

    def initialize(items)
      @items = items
    end

    def each(&blk)
      @items.each do |x|
        blk.call(x)
      end
    end
  end

  nums = Numbers.new([1, 2, 3])
  puts nums.any? { |x| x > 2 }
  puts nums.all? { |x| x > 0 }
  puts nums.none? { |x| x > 5 }
end

gauntlet("include Enumerable reduce") do
  class Numbers
    include Enumerable

    def initialize(items)
      @items = items
    end

    def each(&blk)
      @items.each do |x|
        blk.call(x)
      end
    end
  end

  nums = Numbers.new([1, 2, 3, 4])
  total = nums.reduce(0) { |acc, x| acc + x }
  puts total
end

gauntlet("include Enumerable count with block") do
  class Numbers
    include Enumerable

    def initialize(items)
      @items = items
    end

    def each(&blk)
      @items.each do |x|
        blk.call(x)
      end
    end
  end

  nums = Numbers.new([1, 2, 3, 4, 5])
  puts nums.count { |x| x > 3 }
end

gauntlet("include Enumerable find") do
  class Numbers
    include Enumerable

    def initialize(items)
      @items = items
    end

    def each(&blk)
      @items.each do |x|
        blk.call(x)
      end
    end
  end

  nums = Numbers.new([1, 2, 3, 4, 5])
  found = nums.find { |x| x > 3 }
  puts found
end

gauntlet("include Enumerable min max sum") do
  class Numbers
    include Enumerable

    def initialize(items)
      @items = items
    end

    def each(&blk)
      @items.each do |x|
        blk.call(x)
      end
    end
  end

  nums = Numbers.new([3, 1, 4, 1, 5])
  puts nums.min
  puts nums.max
  puts nums.sum
end


gauntlet("bare predicate method call") do
  class Request
    def initialize(method, path)
      @method = method
      @path = path
    end

    def get?
      @method == 'GET'
    end

    def post?
      @method == 'POST'
    end

    def put?
      @method == 'PUT'
    end

    def safe?
      get?
    end

    def idempotent?
      safe? || put?
    end
  end

  req = Request.new('GET', '/users')
  puts req.get?
  puts req.post?
  puts req.safe?
  puts req.idempotent?

  req2 = Request.new('PUT', '/data')
  puts req2.safe?
  puts req2.idempotent?
end

gauntlet("class method def self.x") do
  class Calculator
    def self.add(a, b)
      a + b
    end

    def self.multiply(a, b)
      a * b
    end
  end

  puts Calculator.add(3, 4)
  puts Calculator.multiply(5, 6)
end

gauntlet("class method with instance methods") do
  class Counter
    def initialize(start)
      @count = start
    end

    def self.zero
      Counter.new(0)
    end

    def value
      @count
    end

    def increment
      @count += 1
    end
  end

  c = Counter.zero
  c.increment
  c.increment
  puts c.value
end

gauntlet("top-level method as receiver in method chain") do
  def get_name
    "hello"
  end

  result = get_name.upcase
  puts result
  puts get_name.length
end

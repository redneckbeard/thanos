class Vehicle
  attr_reader :starting_miles

  def initialize(starting_miles)
    @starting_miles = starting_miles
    @no_reader = "unexported"
  end

  def drive(x)
    @starting_miles += x
  end

  def log
    puts "log was called"
  end

  def mileage
    log
    "#{@starting_miles} miles"
  end
end

class Car < Vehicle
  def log
    puts "it's a different method!"
  end
end

cars = [Car.new(10), Car.new(20), Car.new(30)].map do |car|
  if car.instance_of?(Car) # only here to prove inheritance from Object
    car.drive(100)
  end
  "#{car.mileage}, started at #{car.starting_miles}"
end

puts cars

class Vehicle
  attr_reader :starting_miles
  attr_writer :registration

  def initialize(starting_miles)
    @starting_miles = starting_miles
    @no_reader = "unexported"
  end

  def drive(x)
    @starting_miles += x
  end

  def mileage
    log
    "#{@starting_miles} miles"
  end

  private

  def log
    puts "log was called"
  end
end

class Car < Vehicle
  def drive(x)
    super
    @starting_miles += 1
	end
  # overriding a private method is fine and on the child class is then public
  def log
    puts "it's a different method!"
    super
  end
end

puts [Car.new(10), Car.new(20), Car.new(30)].map do |car|
  if car.instance_of?(Car) # only here to prove inheritance from Object
    car.drive(100)
  end
  car.registration = "XXXXXX"
  "#{car.mileage}, started at #{car.starting_miles}"
end

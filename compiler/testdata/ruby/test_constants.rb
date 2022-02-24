PERMIT_AGE = 16

class CTDriver
  LICENSE_AGE = 18

  def initialize(age)
    @age = age
  end

  def can_drive?
    @age >= LICENSE_AGE
  end
end

class CrossStateCommercialCTDriver < CTDriver
  LICENSE_AGE = 21

  eggplant = 'veg'
end

if CTDriver.new(19).can_drive?
  puts CrossStateCommercialCTDriver::LICENSE_AGE
end

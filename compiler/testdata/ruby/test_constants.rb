PERMIT_AGE = 16

class CTDriver
  LICENSE_AGE = 18
  KIND_MOTORCYCLE = :motorcycle
  KIND_COMMERCIAL = :cdl
  KIND_SCOOTER = :scooter
  LICENSE_KINDS = [KIND_MOTORCYCLE, KIND_COMMERCIAL, KIND_SCOOTER]

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

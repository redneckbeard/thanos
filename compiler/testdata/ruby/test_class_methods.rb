class Parent
  @@family_things = []    # Shared between class and subclasses
  @shared_things  = []    # Specific to this class

  def self.family_things
    @@family_things
  end
  def self.shared_things
    @shared_things
  end

  attr_accessor :my_things
  def initialize
    @my_things = []       # Just for me
  end
  def family_things
    self.class.family_things
  end
  def shared_things
    self.class.shared_things
  end
end

class Child < Parent
  @shared_things = []
end

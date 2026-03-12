gauntlet("nested class definition and instantiation") do
  module Outer
    module Inner
    end
  end

  class Outer::Inner::Item
    def initialize(name)
      @name = name
    end

    attr_reader :name
  end

  item = Outer::Inner::Item.new("test")
  puts item.name
end

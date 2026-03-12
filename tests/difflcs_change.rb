gauntlet("diff-lcs Change basic") do
  module Diff
    module LCS
    end
  end

  class Diff::LCS::Change
    include Comparable

    def initialize(action, position, element)
      @action = action
      @position = position
      @element = element
    end

    VALID_ACTIONS = %w[= - + ! > <].freeze

    attr_reader :action, :position, :element

    def self.valid_action?(action)
      VALID_ACTIONS.include?(action)
    end

    def adding?
      @action == "+"
    end

    def deleting?
      @action == "-"
    end

    def unchanged?
      @action == "="
    end

    def changed?
      @action == "!"
    end

    def finished_a?
      @action == ">"
    end

    def finished_b?
      @action == "<"
    end

    def <=>(other)
      r = @position <=> other.position
      r = @action <=> other.action if r == 0
      r
    end

    def ==(other)
      @action == other.action && @position == other.position && @element == other.element
    end
  end

  c = Diff::LCS::Change.new("+", 5, "hello")
  puts c.action
  puts c.position
  puts c.element
  puts c.adding?
  puts c.deleting?
  puts c.unchanged?
  puts c.changed?
  puts c.finished_a?
  puts c.finished_b?
  puts Diff::LCS::Change.valid_action?("+")
  puts Diff::LCS::Change.valid_action?("x")

  # Comparable via <=>
  a = Diff::LCS::Change.new("+", 3, "foo")
  b = Diff::LCS::Change.new("-", 5, "bar")
  puts a < b
  puts a > b
  puts a == a

end

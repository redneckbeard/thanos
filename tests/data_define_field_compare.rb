gauntlet("Data.define field comparison") do
  Change = Data.define(:action, :position)

  c = Change.new("+", 3)
  puts c.action == "+"
  puts c.action == "-"
  puts c.position == 3
end

gauntlet("Data.define field in method") do
  Entry = Data.define(:kind, :value)

  class Entry
    def adding?
      kind == "+"
    end

    def removing?
      kind == "-"
    end

    def summary
      kind + " at " + value.to_s
    end
  end

  e = Entry.new("+", 42)
  puts e.adding?
  puts e.removing?
  puts e.summary
end

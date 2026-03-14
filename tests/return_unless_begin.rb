gauntlet("return unless with begin/ensure") do
  def do_thing(name)
    result = name

    return result unless name == "go"

    begin
      result = result + " began"
    ensure
      puts "ensured"
    end

    result
  end

  puts do_thing("hello")
  puts do_thing("go")
end

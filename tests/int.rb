gauntlet("Int#abs") do
  #TODO bug, parens required
  puts(-10.abs)
  puts(10.abs)
end

gauntlet("Int#negative?") do
  #TODO bug, parens required
  puts(-10.negative?)
  puts(0.negative?)
  puts(10.negative?)
end

gauntlet("Int#positive?") do
  #TODO bug, parens required
  puts(-10.positive?)
  puts(0.positive?)
  puts(10.positive?)
end

gauntlet("Int#zero?") do
  #TODO bug, parens required
  puts(-10.zero?)
  puts(0.zero?)
  puts(10.zero?)
end

gauntlet("Int#times") do
  10.times do |i|
    puts i
  end
end

gauntlet("Int#upto") do
  10.upto(20) do |i|
    puts i
  end
end

gauntlet("Int#downto") do
  20.downto(10) do |i|
    puts i
  end
end

gauntlet("Int#even?") do
  puts(2.even?)
  puts(3.even?)
  puts(0.even?)
end

gauntlet("Int#odd?") do
  puts(1.odd?)
  puts(2.odd?)
  puts(0.odd?)
end

gauntlet("arithmetic") do
  puts 10 + 3
  puts 10 - 3
  puts 10 * 3
  puts 10 / 3
  puts 10 % 3
  puts 2 ** 10
end

gauntlet("Int#to_s") do
  x = 42
  puts x.to_s
  puts x.to_s + " is the answer"
end

gauntlet("comparison operators") do
  puts 5 > 3
  puts 5 < 3
  puts 5 >= 5
  puts 5 <= 4
  puts 5 == 5
  puts 5 != 3
end

gauntlet("Int#pow") do
  puts 2.pow(10)
  puts 3.pow(3)
end

gauntlet("Int#to_f") do
  x = 42
  puts x.to_f
end

gauntlet("Int#between?") do
  puts 5.between?(1, 10)
  puts 15.between?(1, 10)
  puts 1.between?(1, 10)
  puts 10.between?(1, 10)
end

gauntlet("Int#clamp") do
  puts 5.clamp(1, 10)
  puts(-5.clamp(1, 10))
  puts 15.clamp(1, 10)
end

gauntlet("Int#digits") do
  12345.digits.each { |d| puts d }
end

gauntlet("Int#gcd") do
  puts 12.gcd(8)
  puts 100.gcd(75)
  puts 7.gcd(13)
end

gauntlet("Int#lcm") do
  puts 4.lcm(6)
  puts 12.lcm(8)
  puts 7.lcm(13)
end

gauntlet("Int#step") do
  1.step(10, 3) { |i| puts i }
end

gauntlet("Int#chr") do
  puts 65.chr
  puts 97.chr
  puts 48.chr
end

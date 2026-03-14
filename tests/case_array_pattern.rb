gauntlet("case array pattern") do
  a = true
  b = false
  case [a, b]
  when [true, true]
    puts "both"
  when [true, false]
    puts "first only"
  when [false, true]
    puts "second only"
  else
    puts "neither"
  end

  # Test with expressions
  x = 3
  y = 5
  case [(x < 4), (y < 4)]
  when [true, true]
    puts "both small"
  when [true, false]
    puts "x small"
  when [false, true]
    puts "y small"
  else
    puts "both big"
  end
end

gauntlet("case array pattern with method calls in when body") do
  a_i = 2
  b_j = 7
  m_a = 5
  m_b = 5
  result = ""

  case [(a_i < m_a), (b_j < m_b)]
  when [true, true]
    result = "both in range"
    a_i += 1
    b_j += 1
  when [true, false]
    result = "a only"
    a_i += 1
  when [false, true]
    result = "b only"
    b_j += 1
  when [false, false]
    result = "neither"
  end
  puts result
  puts a_i
  puts b_j
end

gauntlet("case array pattern with if/else in when body") do
  a_i = 2
  b_j = 7
  m_a = 5
  m_b = 5

  while (a_i < m_a) || (b_j < m_b)
    case [(a_i < m_a), (b_j < m_b)]
    when [true, true]
      if a_i < 3
        a_i += 1
      else
        a_i += 2
      end
      b_j += 1
    when [true, false]
      a_i += 1
    when [false, true]
      b_j += 1
    end
  end

  # Match comment
  puts a_i
  puts b_j
end

gauntlet("case array pattern lcs-style") do
  a_i = b_j = 0
  m_a = 3
  m_b = 5

  while (a_i < m_a) || (b_j < m_b)
    case [(a_i < m_a), (b_j < m_b)]
    when [true, true]
      if a_i < 2
        a_i += 1
      else
        a_i += 1
      end
      b_j += 1
    when [true, false]
      a_i += 1
    when [false, true]
      b_j += 1
    end
  end

  # Match
  puts a_i
  puts b_j
end

gauntlet("case array pattern in method with data define") do
  Change = Data.define(:action, :pos_a, :pos_b)

  class Processor
    def process(seq1, seq2)
      a_i = 0
      b_j = 0
      m_a = seq1.length
      m_b = seq2.length
      results = []

      while (a_i < m_a) || (b_j < m_b)
        case [(a_i < m_a), (b_j < m_b)]
        when [true, true]
          results.push(Change.new("!", a_i, b_j))
          a_i += 1
          b_j += 1
        when [true, false]
          results.push(Change.new("-", a_i, b_j))
          a_i += 1
        when [false, true]
          results.push(Change.new("+", a_i, b_j))
          b_j += 1
        end
      end
      results
    end
  end

  p = Processor.new
  changes = p.process([1, 2, 3], [1, 2])
  changes.each do |c|
    puts c.action
  end
end

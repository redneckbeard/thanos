gauntlet("CSV.parse") do
  require 'csv'
  data = CSV.parse("a,b,c\n1,2,3\n4,5,6")
  data.each do |row|
    puts row.join(" ")
  end
end

gauntlet("CSV.foreach") do
  require 'csv'
  File.write("/tmp/thanos_csv_test.csv", "name,age\nAlice,30\nBob,25\n")
  CSV.foreach("/tmp/thanos_csv_test.csv") do |row|
    puts row.join(" ")
  end
end

gauntlet("CSV.generate") do
  require 'csv'
  result = CSV.generate do |csv|
    csv << ["name", "age"]
    csv << ["Alice", "30"]
  end
  puts result
end

gauntlet("CSV.read") do
  require 'csv'
  File.write("/tmp/thanos_csv_read.csv", "x,y\n1,2\n3,4\n")
  rows = CSV.read("/tmp/thanos_csv_read.csv")
  rows.each do |row|
    puts row.join(" ")
  end
end

gauntlet("CSV.open write") do
  require 'csv'
  CSV.open("/tmp/thanos_csv_out.csv", "w") do |csv|
    csv << ["hello", "world"]
    csv << ["foo", "bar"]
  end
  puts File.read("/tmp/thanos_csv_out.csv")
end

gauntlet("CSV.parse with headers") do
  require 'csv'
  table = CSV.parse("name,age\nAlice,30\nBob,25", headers: true)
  table.each do |row|
    puts row["name"]
    puts row["age"]
  end
end

gauntlet("CSV.read with headers") do
  require 'csv'
  File.write("/tmp/thanos_csv_hdr.csv", "x,y\n1,2\n3,4\n")
  table = CSV.read("/tmp/thanos_csv_hdr.csv", headers: true)
  puts table.headers.join(" ")
  table.each { |row| puts row["x"] }
end

gauntlet("CSV.foreach with headers") do
  require 'csv'
  File.write("/tmp/thanos_csv_feh.csv", "name,score\nAlice,95\nBob,87\n")
  CSV.foreach("/tmp/thanos_csv_feh.csv", headers: true) do |row|
    puts row["name"]
  end
end

gauntlet("CSV.parse with col_sep") do
  require 'csv'
  data = CSV.parse("a\tb\tc\n1\t2\t3", col_sep: "\t")
  data.each { |row| puts row.join(" ") }
end

gauntlet("CSV::Row methods") do
  require 'csv'
  table = CSV.parse("name,age,city\nAlice,30,NYC\n", headers: true)
  row = table[0]
  puts row["name"]
  puts row.headers.join(",")
  puts row.fields.join(",")
end

gauntlet("CSV::Table bracket and length") do
  require 'csv'
  table = CSV.parse("a,b\n1,2\n3,4\n5,6", headers: true)
  puts table.length
  puts table[0]["a"]
  puts table[2]["b"]
end

gauntlet("CSV.parse with headers and col_sep") do
  require 'csv'
  table = CSV.parse("name\tage\nAlice\t30\nBob\t25", headers: true, col_sep: "\t")
  table.each do |row|
    puts row["name"]
  end
end

gauntlet("CSV.read with col_sep") do
  require 'csv'
  File.write("/tmp/thanos_csv_tsv.csv", "a\tb\n1\t2\n3\t4\n")
  data = CSV.read("/tmp/thanos_csv_tsv.csv", col_sep: "\t")
  data.each { |row| puts row.join(" ") }
end

gauntlet("CSV.foreach with col_sep") do
  require 'csv'
  File.write("/tmp/thanos_csv_ftsv.csv", "x\ty\n10\t20\n30\t40\n")
  CSV.foreach("/tmp/thanos_csv_ftsv.csv", col_sep: "\t") do |row|
    puts row.join(" ")
  end
end

gauntlet("CSV::Table size") do
  require 'csv'
  table = CSV.parse("a,b\n1,2\n3,4", headers: true)
  puts table.size
end

gauntlet("CSV::Row to_h") do
  require 'csv'
  table = CSV.parse("name,age,city\nAlice,30,NYC\n", headers: true)
  h = table[0].to_h
  puts h["name"]
  puts h["age"]
  puts h["city"]
end

gauntlet("CSV::Table to_a") do
  require 'csv'
  table = CSV.parse("a,b\n1,2\n3,4", headers: true)
  arr = table.to_a
  puts arr.length
  puts arr[0].join(",")
  puts arr[1].join(",")
  puts arr[2].join(",")
end

gauntlet("CSV::Row to_csv") do
  require 'csv'
  table = CSV.parse("name,age\nAlice,30\n", headers: true)
  puts table[0].to_csv
end

gauntlet("CSV::Table to_csv") do
  require 'csv'
  table = CSV.parse("name,age\nAlice,30\nBob,25", headers: true)
  puts table.to_csv
end

gauntlet("CSV::Row delete") do
  require 'csv'
  table = CSV.parse("name,age,city\nAlice,30,NYC\n", headers: true)
  row = table[0]
  val = row.delete("age")
  puts val[0]
  puts val[1]
  puts row.headers.join(",")
  puts row.fields.join(",")
end

gauntlet("CSV::Row []=") do
  require 'csv'
  table = CSV.parse("name,age\nAlice,30\n", headers: true)
  row = table[0]
  row["age"] = "31"
  puts row["age"]
  row["city"] = "NYC"
  puts row["city"]
  puts row.headers.join(",")
end

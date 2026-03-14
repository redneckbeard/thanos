gauntlet("File#each") do
  File.new("compiler/testdata/input/millennials.txt").each do |line|
    puts line.gsub(/[mM]illennial(?<plural>s)?/, 'Snake Person\k<plural>')
  end
end

gauntlet("File.basename") do
  puts File.basename("/usr/local/bin/ruby")
  puts File.basename("/home/user/file.txt")
  puts File.basename("file.rb")
end

gauntlet("File.dirname") do
  puts File.dirname("/usr/local/bin/ruby")
  puts File.dirname("/home/user/file.txt")
  puts File.dirname("file.rb")
end

gauntlet("File.extname") do
  puts File.extname("test.rb")
  puts File.extname("archive.tar.gz")
  puts File.extname("Makefile")
end

gauntlet("File.exist?") do
  puts File.exist?("compiler/testdata/input/millennials.txt")
  puts File.exist?("nonexistent_file_12345.txt")
end

gauntlet("File.exist?") do
  puts File.exist?("compiler/testdata/input/millennials.txt")
  puts File.exist?("nonexistent_file_98765.txt")
end

gauntlet("File.directory?") do
  puts File.directory?("compiler")
  puts File.directory?("compiler/testdata/input/millennials.txt")
  puts File.directory?("nonexistent_dir_12345")
end

gauntlet("File.write and File.read") do
  path = "/tmp/thanos_test_rw.txt"
  File.write(path, "hello world")
  content = File.read(path)
  puts content
  File.delete(path)
end

gauntlet("File.size") do
  path = "/tmp/thanos_test_size.txt"
  File.write(path, "12345")
  puts File.size(path)
  File.delete(path)
end

gauntlet("File.open with block") do
  path = "/tmp/thanos_test_open.txt"
  File.write(path, "line one\nline two\nline three\n")
  lines = []
  File.open(path) do |f|
    f.each_line do |line|
      lines << line
    end
  end
  lines.each { |l| puts l }
  File.delete(path)
end

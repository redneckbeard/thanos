gauntlet("Zlib gzip and gunzip roundtrip") do
  require 'zlib'
  original = "hello world from zlib"
  compressed = Zlib.gzip(original)
  decompressed = Zlib.gunzip(compressed)
  puts decompressed
end

gauntlet("Zlib deflate and inflate roundtrip") do
  require 'zlib'
  original = "test string for deflate"
  compressed = Zlib.deflate(original)
  decompressed = Zlib.inflate(compressed)
  puts decompressed
end

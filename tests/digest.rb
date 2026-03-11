gauntlet("Digest::SHA256.hexdigest") do
  require 'digest'
  puts Digest::SHA256.hexdigest("hello")
end

gauntlet("Digest::MD5.hexdigest") do
  require 'digest'
  puts Digest::MD5.hexdigest("hello")
end

gauntlet("Digest::SHA1.hexdigest") do
  require 'digest'
  puts Digest::SHA1.hexdigest("hello")
end

gauntlet("Digest::SHA512.hexdigest") do
  require 'digest'
  puts Digest::SHA512.hexdigest("hello")
end

gauntlet("Digest::SHA384.hexdigest") do
  require 'digest'
  puts Digest::SHA384.hexdigest("hello")
end

gauntlet("Digest::SHA256.base64digest") do
  require 'digest'
  puts Digest::SHA256.base64digest("hello")
end

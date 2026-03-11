#!/usr/bin/env ruby
#
# Scans Ruby stdlib modules for potential Tier 1 facade candidates.
# A good candidate has:
#   - Module-level (singleton) methods only (no instance state)
#   - Simple argument types (string, numeric)
#   - A clear Go stdlib equivalent
#
# Usage: ruby scripts/find_facade_candidates.rb

# Stdlib modules that need `require` and are worth inspecting.
# Excludes: modules needing C extensions, metaprogramming-heavy, or
# already handled natively by thanos (Set, Comparable, Enumerable, Time).
CANDIDATES = %w[
  base64
  cgi
  csv
  digest
  erb
  fileutils
  json
  logger
  matrix
  net/http
  open-uri
  pathname
  pp
  securerandom
  shellwords
  singleton
  tempfile
  uri
  yaml
  zlib
]

puts "=" * 70
puts "Ruby stdlib facade candidate scan"
puts "=" * 70
puts

CANDIDATES.each do |lib|
  begin
    require lib
  rescue LoadError => e
    puts "SKIP  #{lib} (LoadError: #{e.message})"
    puts
    next
  end

  # Find the top-level constant(s) this library defines
  mod_name = lib.split('/').map { |p| p.split(/[-_]/).map(&:capitalize).join }.join('::')

  # Try common capitalization patterns
  mod = nil
  [mod_name, mod_name.upcase, lib.capitalize, lib.upcase].each do |guess|
    begin
      mod = Object.const_get(guess)
      mod_name = guess
      break
    rescue NameError
      next
    end
  end

  unless mod
    puts "SKIP  #{lib} (could not resolve module constant)"
    puts
    next
  end

  # Get singleton methods (module-level methods)
  singleton_methods = if mod.is_a?(Module)
    mod.singleton_methods(false).sort
  else
    []
  end

  # Get instance methods if it's a class
  instance_methods = if mod.is_a?(Class)
    mod.public_instance_methods(false).sort
  else
    []
  end

  next if singleton_methods.empty? && instance_methods.empty?

  # Score: more singleton methods = better Tier 1 candidate
  tier1_score = singleton_methods.length
  tier1_score = 0 if instance_methods.length > singleton_methods.length * 2

  label = case tier1_score
          when 0 then "UNLIKELY"
          when 1..3 then "MAYBE"
          when 4..10 then "GOOD"
          else "GREAT"
          end

  puts "#{label.ljust(10)} #{lib} (as #{mod_name})"
  unless singleton_methods.empty?
    puts "  Module methods (#{singleton_methods.length}):"
    singleton_methods.each do |m|
      arity = mod.method(m).arity
      params = mod.method(m).parameters.map { |kind, name| "#{kind}:#{name}" }.join(", ")
      puts "    .#{m}(#{params}) arity=#{arity}"
    end
  end
  unless instance_methods.empty?
    puts "  Instance methods (#{instance_methods.length}):"
    instance_methods.each do |m|
      puts "    ##{m}"
    end
  end
  puts
end

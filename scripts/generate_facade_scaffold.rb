#!/usr/bin/env ruby
#
# Generates a facade scaffold for a Ruby stdlib module.
#
# Given a Ruby library name, this script:
#   1. Introspects every public method (module + instance on key classes)
#   2. Classifies each method's signature complexity
#   3. Emits a scaffold: JSON facade + Go runtime stubs + gauntlet test stubs
#
# Usage:
#   ruby scripts/generate_facade_scaffold.rb <library_name> [ClassName ...]
#
# Examples:
#   ruby scripts/generate_facade_scaffold.rb shellwords
#   ruby scripts/generate_facade_scaffold.rb uri URI URI::Generic URI::HTTP
#   ruby scripts/generate_facade_scaffold.rb fileutils FileUtils
#
# The scaffold is written to scratch/facades/<library_name>/ and is meant
# to be reviewed and edited by a human before moving into place.

require 'json'
require 'fileutils'
require 'set'

# Known Ruby→Go stdlib mappings. Keys are "Module.method" or "Module#method".
# This is the closest thing we have to a deterministic step — when a mapping
# exists here, the scaffold can emit a complete Tier 1 entry with no human input.
#
# Format: { ruby_qual => { call: "go.path", returns: "thanos_type", args: [...] } }
GO_STDLIB_MAP = {
  # Shellwords
  "Shellwords.escape"      => { call: "shims.ShellEscape",   returns: "string" },
  "Shellwords.shellescape" => { call: "shims.ShellEscape",   returns: "string" },
  "Shellwords.split"       => { call: "shims.ShellSplit",    returns: "[]string" },
  "Shellwords.shellsplit"  => { call: "shims.ShellSplit",    returns: "[]string" },
  "Shellwords.shellwords"  => { call: "shims.ShellSplit",    returns: "[]string" },
  "Shellwords.join"        => { call: "shims.ShellJoin",     returns: "string" },
  "Shellwords.shelljoin"   => { call: "shims.ShellJoin",     returns: "string" },

  # URI
  "URI.parse"              => { call: "shims.URIParse",      returns: "URI" },
  "URI.encode_www_form_component" => { call: "url.QueryEscape", returns: "string", imports: ["net/url"] },
  "URI.decode_www_form_component" => { call: "url.QueryUnescape", returns: "string", imports: ["net/url"], ignore_error: true },

  # FileUtils
  "FileUtils.mkdir_p"     => { call: "os.MkdirAll", returns: "nil", args: [{}, { default: "0o755" }], ignore_error: true },
  "FileUtils.rm"           => { call: "os.Remove",   returns: "nil", ignore_error: true },
  "FileUtils.rm_rf"        => { call: "os.RemoveAll", returns: "nil", ignore_error: true },
  "FileUtils.cp"           => { call: "shims.CopyFile", returns: "nil" },
  "FileUtils.mv"           => { call: "os.Rename",   returns: "nil", ignore_error: true },
  "FileUtils.touch"        => { call: "shims.Touch",  returns: "nil" },
}

# Known return types by method name pattern (heuristic).
# These are guesses based on Ruby naming conventions.
RETURN_TYPE_HEURISTICS = {
  /\?$/          => "bool",      # predicate methods return bool
  /^to_s$/       => "string",
  /^to_i$/       => "int",
  /^to_f$/       => "float",
  /^to_a$/       => "[]string",  # often, not always
  /^to_h$/       => "{string: string}",
  /^length$/     => "int",
  /^size$/       => "int",
  /^count$/      => "int",
  /^empty\?$/    => "bool",
  /^include\?$/  => "bool",
  /^index$/      => "int",
  /^split$/      => "[]string",
  /^join$/       => "string",
  /^inspect$/    => "string",
  /^hash$/       => "int",
}

# Methods that are known aliases of each other in Ruby stdlib.
# Used to detect and emit alias entries instead of duplicate implementations.
KNOWN_ALIASES = {
  "Shellwords" => {
    "shellescape" => "escape",
    "shellsplit" => "split",
    "shellwords" => "split",
    "shelljoin" => "join",
  },
}

# Classification of a single method
MethodInfo = Struct.new(
  :name,           # Ruby method name
  :kind,           # :module or :instance
  :owner,          # module/class name that owns this method
  :arity,          # Ruby arity
  :params,         # [[kind, name], ...] from Method#parameters
  :tier,           # :tier1, :tier2_shim, :tier2_type, :tier3
  :reason,         # why this tier was chosen
  :has_block,      # takes a block?
  :has_kwargs,     # takes keyword args?
  :has_rest,       # takes *args?
  :go_mapping,     # entry from GO_STDLIB_MAP if found
  :guessed_return, # heuristic return type guess
  :alias_of,       # if this is an alias, the canonical method name
  keyword_init: true
)

def classify_method(mod, method_name, kind, owner_name)
  m = case kind
      when :module then mod.method(method_name)
      when :instance then mod.instance_method(method_name)
      end

  params = m.parameters
  param_kinds = params.map(&:first)

  has_block   = param_kinds.include?(:block)
  has_kwargs  = param_kinds.any? { |k| %i[key keyreq keyrest].include?(k) }
  has_rest    = param_kinds.include?(:rest)
  req_count   = param_kinds.count(:req)
  opt_count   = param_kinds.count(:opt)

  # Check for known Go mapping
  sep = kind == :module ? "." : "#"
  qual = "#{owner_name}#{sep}#{method_name}"
  go_mapping = GO_STDLIB_MAP[qual]

  # Check for known alias
  alias_of = nil
  if (aliases = KNOWN_ALIASES[owner_name])
    alias_of = aliases[method_name.to_s]
  end

  # Guess return type from method name
  guessed_return = nil
  RETURN_TYPE_HEURISTICS.each do |pattern, ret|
    if method_name.to_s.match?(pattern)
      guessed_return = ret
      break
    end
  end

  # Decision tree
  tier = :tier1
  reason = "static args, static return"

  if go_mapping
    # Known mapping overrides — we know exactly what to emit
    tier = :tier1
    reason = "known Go mapping"
  elsif alias_of
    tier = :tier1
    reason = "alias of #{alias_of}"
  elsif has_block && has_kwargs
    tier = :tier3
    reason = "block + kwargs interaction"
  elsif has_block
    tier = :tier3
    reason = "block changes semantics"
  elsif has_kwargs
    tier = :tier3
    reason = "kwargs may change behavior"
  elsif has_rest
    tier = :tier2_shim
    reason = "variadic args need Go wrapper"
  elsif opt_count > 0
    tier = :tier2_shim
    reason = "optional args need Go wrapper or overloaded functions"
  end

  MethodInfo.new(
    name: method_name.to_s,
    kind: kind,
    owner: owner_name,
    arity: m.arity,
    params: params,
    tier: tier,
    reason: reason,
    has_block: has_block,
    has_kwargs: has_kwargs,
    has_rest: has_rest,
    go_mapping: go_mapping,
    guessed_return: guessed_return,
    alias_of: alias_of,
  )
end

def introspect_module(mod, mod_name)
  methods = []

  # Module-level (singleton) methods
  mod.singleton_methods(false).sort.each do |m|
    methods << classify_method(mod, m, :module, mod_name)
  end

  # Instance methods if it's a class
  if mod.is_a?(Class)
    mod.public_instance_methods(false).sort.each do |m|
      methods << classify_method(mod, m, :instance, mod_name)
    end
  end

  methods
end

def format_params(params)
  params.map { |kind, name| "#{kind}:#{name}" }.join(", ")
end

def ruby_to_go_method_name(ruby_name)
  # Convert Ruby method names to Go-style: foo_bar → FooBar, empty? → IsEmpty
  name = ruby_name.dup
  name = "Is#{name.chomp('?').split('_').map(&:capitalize).join}" if name.end_with?('?')
  name = "#{name.chomp('!').split('_').map(&:capitalize).join}InPlace" if name.end_with?('!')
  name = name.split('_').map(&:capitalize).join unless name.include?('?') || name.include?('!')
  name
end

def generate_json_facade(lib_name, mod_name, all_methods)
  mod_methods = all_methods.select { |m| m.kind == :module && %i[tier1 tier2_shim].include?(m.tier) }
  instance_methods = all_methods.select { |m| m.kind == :instance && %i[tier1 tier2_shim].include?(m.tier) }

  # Collect all imports needed
  imports = Set.new(["github.com/redneckbeard/thanos/shims"])
  all_methods.each do |m|
    next unless m.go_mapping && m.go_mapping[:imports]
    m.go_mapping[:imports].each { |i| imports << i }
  end

  facade = {
    lib_name => {
      "go_imports" => imports.to_a.sort,
      "modules" => {},
      "types" => {},
    }
  }

  # Track which methods are canonical vs aliases
  canonical_methods = {}

  unless mod_methods.empty?
    methods_hash = {}
    mod_methods.each do |m|
      if m.alias_of
        # Skip aliases for now — we'll emit them after canonicals
        next
      end

      entry = if m.go_mapping
        gm = m.go_mapping
        e = { "call" => [gm[:call]], "returns" => gm[:returns] }
        e["ignore_error"] = true if gm[:ignore_error]
        e["args"] = gm[:args].map { |a| a.is_a?(Hash) ? a : {} } if gm[:args]
        e
      else
        ret = m.guessed_return || "string"
        {
          "call" => ["shims.#{mod_name}#{ruby_to_go_method_name(m.name)}"],
          "returns" => ret,
          "_TODO" => ret == "string" && !m.guessed_return ? "verify return type" : nil,
        }.compact
      end

      methods_hash[m.name] = entry
      canonical_methods[m.name] = true
    end

    # Now add aliases
    mod_methods.each do |m|
      next unless m.alias_of && canonical_methods[m.alias_of]
      methods_hash[m.name] = methods_hash[m.alias_of]
    end

    facade[lib_name]["modules"][mod_name] = { "methods" => methods_hash }
  end

  # Instance methods → types section
  unless instance_methods.empty?
    type_methods = {}
    instance_methods.each do |m|
      ret = if m.go_mapping
        m.go_mapping[:returns]
      else
        m.guessed_return || "string"
      end

      go_method = if m.go_mapping
        m.go_mapping[:call].split('.').last
      else
        ruby_to_go_method_name(m.name)
      end

      type_methods[m.name] = { "call" => go_method, "returns" => ret }
    end

    owner = instance_methods.first.owner
    facade[lib_name]["types"][owner] = {
      "go_type" => "*#{lib_name}.#{owner.split('::').last}",
      "methods" => type_methods,
      "_TODO" => "verify go_type and create Go struct",
    }
  end

  facade
end

def generate_go_shim_stubs(lib_name, mod_name, methods)
  shim_methods = methods.select { |m| %i[tier1 tier2_shim].include?(m.tier) && m.kind == :module }
  return nil if shim_methods.empty?

  lines = ["package shims", ""]
  shim_methods.each do |m|
    go_name = "#{mod_name}#{ruby_to_go_method_name(m.name)}"
    req_params = m.params.select { |k, _| k == :req }
    go_params = req_params.map { |_, name| "#{name} string" }.join(", ")
    lines << "// #{go_name} implements Ruby's #{mod_name}.#{m.name}"
    lines << "// TODO: implement — placeholder returns empty string"
    lines << "func #{go_name}(#{go_params}) string {"
    lines << "\treturn \"\""
    lines << "}"
    lines << ""
  end

  lines.join("\n")
end

def generate_tier3_stubs(lib_name, mod_name, methods)
  tier3_methods = methods.select { |m| m.tier == :tier3 }
  return nil if tier3_methods.empty?

  lines = [
    "package #{lib_name}",
    "",
    "import (",
    "\t\"github.com/redneckbeard/thanos/bst\"",
    "\t\"github.com/redneckbeard/thanos/types\"",
    ")",
    "",
    "func init() {",
  ]

  tier3_methods.each do |m|
    lines << "\t// #{mod_name}.#{m.name} — #{m.reason}"
    lines << "\t// Params: #{format_params(m.params)}"

    kwargs = m.params.select { |k, _| %i[key keyreq].include?(k) }
    unless kwargs.empty?
      lines << "\t// KwargsSpec needed:"
      kwargs.each { |_, name| lines << "\t//   {Name: \"#{name}\", Type: types.StringType}," }
    end

    lines << "\t// types.#{mod_name}Class.Def(\"#{m.name}\", types.MethodSpec{"
    lines << "\t// \t// TODO: implement ReturnType, TransformAST"
    lines << "\t// })"
    lines << ""
  end

  lines << "}"
  lines.join("\n")
end

def generate_gauntlet_stubs(lib_name, mod_name, methods)
  lines = []
  methods.each do |m|
    qual = m.kind == :module ? "#{mod_name}.#{m.name}" : "#{mod_name}##{m.name}"
    lines << "# gauntlet(\"#{qual}\") do"
    lines << "#   require '#{lib_name}'"
    lines << "#   # TODO: write test"
    lines << "#   # Tier: #{m.tier} (#{m.reason})"
    lines << "#   # Params: #{format_params(m.params)}"
    lines << "# end"
    lines << ""
  end
  lines.join("\n")
end

def generate_report(mod_name, methods)
  lines = []
  lines << ""
  lines << "=" * 70
  lines << "FACADE SCAFFOLD REPORT: #{mod_name}"
  lines << "=" * 70

  by_tier = methods.group_by(&:tier)

  { tier1: "Tier 1 (pure JSON)", tier2_shim: "Tier 2 (shim)",
    tier2_type: "Tier 2 (declarative type)", tier3: "Tier 3 (Go init)" }.each do |tier, label|
    next unless by_tier[tier]
    lines << ""
    lines << "#{label}: #{by_tier[tier].length} methods"
    by_tier[tier].each do |m|
      prefix = m.kind == :module ? "." : "#"
      status = if m.go_mapping
        "MAPPED"
      elsif m.alias_of
        "ALIAS→#{m.alias_of}"
      elsif m.guessed_return
        "GUESSED(→#{m.guessed_return})"
      else
        "NEEDS WORK"
      end
      lines << "  [#{status.ljust(20)}] #{prefix}#{m.name}(#{format_params(m.params)}) — #{m.reason}"
    end
  end

  total = methods.length
  tier1_count = (by_tier[:tier1] || []).length
  mapped_count = methods.count { |m| m.go_mapping }
  alias_count = methods.count { |m| m.alias_of && !m.go_mapping }
  guessed_count = methods.count { |m| m.guessed_return && !m.go_mapping && !m.alias_of }
  automated = mapped_count + alias_count
  needs_work = total - automated

  json_pct = total > 0 ? (tier1_count * 100.0 / total).round(0) : 0

  lines << ""
  lines << "SUMMARY: #{total} methods total"
  lines << "  #{mapped_count} have known Go mappings (fully automated)"
  lines << "  #{alias_count} are aliases (automated once canonical is done)"
  lines << "  #{guessed_count} have heuristic return type guesses (review needed)"
  lines << "  #{needs_work} need manual mapping work"
  lines << ""
  lines << "  #{tier1_count} pure JSON (#{json_pct}%)"
  lines << "  #{(by_tier[:tier2_shim] || []).length} need Go shims"
  lines << "  #{(by_tier[:tier3] || []).length} need Tier 3 (custom transforms)"
  lines << ""

  if mapped_count == total
    lines << "VERDICT: Fully automatable — all methods have known Go mappings"
  elsif json_pct >= 80
    lines << "VERDICT: Excellent facade candidate — mostly Tier 1"
  elsif json_pct >= 50
    lines << "VERDICT: Good candidate — majority Tier 1, some shims needed"
  elsif tier1_count > 0
    lines << "VERDICT: Mixed — partial JSON coverage, significant Go work needed"
  else
    lines << "VERDICT: Complex — all methods need Go implementation"
  end

  lines.join("\n")
end

# --- Main ---

if ARGV.empty?
  $stderr.puts "Usage: ruby scripts/generate_facade_scaffold.rb <library> [ClassName ...]"
  exit 1
end

lib_name = ARGV[0]
extra_classes = ARGV[1..]

begin
  require lib_name
rescue LoadError => e
  $stderr.puts "Cannot load '#{lib_name}': #{e.message}"
  exit 1
end

# Resolve the main module constant.
# If the first extra_class is a valid constant, use it as the primary module.
mod = nil
mod_name = nil

if extra_classes.any?
  begin
    mod = Object.const_get(extra_classes[0])
    mod_name = extra_classes.shift
  rescue NameError
    # Fall through to guessing
  end
end

unless mod
  mod_name = lib_name.split('/').map { |p| p.split(/[-_]/).map(&:capitalize).join }.join('::')
  guesses = [mod_name, mod_name.upcase, lib_name.capitalize, lib_name.upcase]
  guesses << lib_name.gsub(/(?:^|_)([a-z])/) { $1.upcase }
  %w[utils file net http https uri ftp smtp pop imap ldap].each do |word|
    if lib_name.include?(word) && lib_name != word
      guesses << lib_name.gsub(word, word.capitalize).then { |s| s[0].upcase + s[1..] }
    end
  end
  guesses.uniq.each do |guess|
    begin
      mod = Object.const_get(guess)
      mod_name = guess
      break
    rescue NameError
      next
    end
  end
end

unless mod
  $stderr.puts "Could not resolve module constant for '#{lib_name}'"
  $stderr.puts "Hint: pass the main class/module name explicitly, e.g.:"
  $stderr.puts "  ruby #{$0} #{lib_name} Net::HTTP Net::HTTPResponse"
  exit 1
end

# Collect all methods
all_methods = introspect_module(mod, mod_name)

# Also introspect extra classes
extra_classes.each do |cls_name|
  begin
    cls = Object.const_get(cls_name)
    all_methods.concat(introspect_module(cls, cls_name))
  rescue NameError
    $stderr.puts "Warning: could not resolve #{cls_name}"
  end
end

# Generate scaffold
safe_name = lib_name.gsub('/', '_')
out_dir = "scratch/facades/#{safe_name}"
FileUtils.mkdir_p(out_dir)

# JSON facade
json = generate_json_facade(lib_name, mod_name, all_methods)
File.write("#{out_dir}/#{safe_name}.json", JSON.pretty_generate(json) + "\n")

# Go shim stubs
if (shim = generate_go_shim_stubs(lib_name, mod_name, all_methods))
  File.write("#{out_dir}/shim_stubs.go.stub", shim)
end

# Tier 3 stubs
if (t3 = generate_tier3_stubs(lib_name, mod_name, all_methods))
  File.write("#{out_dir}/types_stubs.go.stub", t3)
end

# Gauntlet test stubs
tests = generate_gauntlet_stubs(lib_name, mod_name, all_methods)
File.write("#{out_dir}/tests.rb", tests)

# Report
report = generate_report(mod_name, all_methods)
puts report

puts ""
puts "Scaffold written to #{out_dir}/"
puts "  #{lib_name}.json      — JSON facade (Tier 1 methods only)"
puts "  shim_stubs.go.stub   — Go function stubs for Tier 1/2 methods"
puts "  types_stubs.go.stub  — Tier 3 init() skeleton (if applicable)"
puts "  tests.rb        — Gauntlet test stubs (all methods)"

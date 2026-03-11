# Thanos Roadmap

Current status: **418 gauntlet tests** passing, 0 failures.

## Where We Are

### Implemented Language Features
- Classes with inheritance, constructors, instance variables, attr_accessor/reader/writer
- Class methods (`def self.x`) — compiled as standalone Go functions
- Modules (top-level, as namespaces for classes and constants)
- Methods with default args, splat (`*args`), double-splat (`**kwargs`), blocks, `yield`
- Control flow: if/elsif/else, unless, while, until, case/when, for-in, break, next, return
- Postfix conditionals (`x if cond`, `x unless cond`, `x while cond`, `x until cond`)
- Ternary operator (`condition ? true_val : false_val`)
- `next` and `break` in loops and blocks, `next(value)` in map blocks
- String interpolation, regex literals, symbols (as strings)
- String `%` formatting (`"Hello %s" % name`)
- Ranges (inclusive and exclusive) with iteration methods (each, map, select, reduce, etc.)
- Multiple assignment and swap (`a, b = b, a`)
- LHS splat (`first, *rest = arr`)
- Operators: arithmetic, comparison, logical, `=~`, `||=`, `&&=`
- Scoped nil/Optional: nil in arrays, `compact`, `||` default, `||=`, `&.` safe navigation
- Destructured block params `|(k, v)|`
- `&:symbol` shorthand (`array.map(&:to_s)`)
- Constants, class variables (`@@var`), global variables (`$var`), `super`, scope access (`::`)
- Hash contextual nil: `h[key] || default` with compiler warnings, `Hash#fetch`
- Exception handling: `begin/rescue/ensure/end`, `raise`, typed rescue clauses
- Lambda/Proc literals (`->`, `.call()`)
- Struct.new with member methods
- OrderedMap: hashes preserve insertion order via `stdlib.OrderedMap`
- Hash.new with default values and block form (including array accumulation pattern)
- Hash-accessed array mutation: `h["key"] << val` compiles correctly for both regular and DefaultHash
- In-place mutating methods on Array and Hash
- Module mixins via registry: `include Comparable`, `include Enumerable`
- `alias` keyword and `alias_method` class method
- Open classes (reopen class to add methods)
- Multi-file support via `require_relative`
- Comment preservation (Ruby `#` → Go `//`)

### Type System
- Type inference from literals, method calls, and block returns
- Composite types: Array, Hash, Set, Optional, Tuple, Range
- Empty array/hash type inference from subsequent operations
- Block param type refinement
- Inherited method resolution
- Deferred string interpolation format verbs
- Hash ordering analysis (OrderedMap → native map lowering when safe)

### Implemented Methods (by type)

**Array** (81 methods):
`&`, `|`, `+`, `-`, `<<`, `all?`, `any?`, `clear`, `combination`, `compact`, `compact!`, `collect!`, `concat`, `count`, `delete`, `delete_at`, `detect`, `dig`, `drop`, `drop_while`, `each`, `each_cons`, `each_index`, `each_slice`, `each_with_index`, `each_with_object`, `empty?`, `fetch`, `fill`, `find`, `find_index`, `first`, `flat_map`, `flatten`, `flatten!`, `group_by`, `include?`, `index`, `insert`, `join`, `last`, `length`, `map`, `map!`, `max`, `max_by`, `member?`, `min`, `min_by`, `none?`, `one?`, `partition`, `permutation`, `pop`, `product`, `push`, `reduce`, `reject`, `reject!`, `reverse`, `reverse!`, `reverse_each`, `rindex`, `rotate`, `sample`, `select`, `select!`, `shift`, `shuffle`, `sort`, `sort!`, `sort_by`, `sort_by!`, `sum`, `take`, `take_while`, `tally`, `transpose`, `uniq`, `uniq!`, `unshift`, `values_at`, `zip`

**String** (86 methods):
`!=`, `%`, `*`, `+`, `<`, `<<`, `<=`, `<=>`, `==`, `=~`, `>`, `>=`, `between?`, `bytes`, `bytesize`, `capitalize`, `capitalize!`, `casecmp?`, `center`, `chars`, `chomp`, `chomp!`, `chop`, `chop!`, `chr`, `clear`, `codepoints`, `concat`, `count`, `delete`, `delete!`, `delete_prefix`, `delete_prefix!`, `delete_suffix`, `delete_suffix!`, `downcase`, `downcase!`, `each_char`, `each_line`, `empty?`, `encode`, `end_with?`, `freeze`, `gsub`, `gsub!`, `hex`, `include?`, `index`, `insert`, `length`, `lines`, `ljust`, `lstrip`, `lstrip!`, `match`, `match?`, `oct`, `ord`, `partition`, `prepend`, `replace`, `reverse`, `rindex`, `rjust`, `rpartition`, `rstrip`, `rstrip!`, `scan`, `size`, `split`, `squeeze`, `squeeze!`, `start_with?`, `strip`, `strip!`, `sub`, `sub!`, `succ`, `swapcase`, `swapcase!`, `to_f`, `to_i`, `tr`, `upcase`, `upcase!`, `upto`

**Hash** (47 methods):
`[]`, `[]=`, `all?`, `any?`, `clear`, `count`, `delete`, `delete_if`, `dig`, `each`, `each_key`, `each_value`, `each_with_index`, `each_with_object`, `empty?`, `fetch`, `filter`, `flat_map`, `has_key?`, `has_value?`, `invert`, `key`, `key?`, `keys`, `length`, `map`, `max_by`, `merge`, `merge!`, `min_by`, `new`, `none?`, `reduce`, `reject`, `reject!`, `select!`, `shift`, `size`, `sort_by`, `sum`, `to_a`, `transform_keys`, `transform_values`, `transform_values!`, `value?`, `values`, `values_at`

**Integer** (23 methods):
`&`, `<<`, `>>`, `[]`, `^`, `between?`, `chr`, `clamp`, `digits`, `downto`, `even?`, `gcd`, `integer?`, `lcm`, `odd?`, `pow`, `step`, `times`, `to_f`, `to_i`, `to_s`, `upto`, `|`

**Range** (21 methods):
`===`, `any?`, `detect`, `each`, `each_with_index`, `find`, `first`, `include?`, `inject`, `last`, `length`, `map`, `max`, `min`, `none?`, `reduce`, `reject`, `select`, `size`, `sum`, `to_a`

**Time** (21 methods):
`+`, `-`, `<`, `==`, `>`, `day`, `hour`, `initialize`, `min`, `month`, `now`, `sec`, `strftime`, `to_f`, `to_i`, `to_s`, `utc`, `utc?`, `wday`, `yday`, `year`

**Numeric** (17 methods — shared base for Integer/Float):
`!=`, `%`, `*`, `**`, `+`, `-`, `/`, `<`, `<=`, `<=>`, `==`, `>`, `>=`, `abs`, `negative?`, `positive?`, `zero?`

**File** (16 methods):
`<<`, `basename`, `close`, `delete`, `directory?`, `dirname`, `each`, `exist?`, `extname`, `initialize`, `open`, `path`, `puts`, `read`, `size`, `write`

**Float** (12 methods):
`abs`, `between?`, `ceil`, `clamp`, `finite?`, `floor`, `infinite?`, `nan?`, `round`, `to_i`, `to_s`, `zero?`

**Object** (10 methods):
`&&`, `==`, `instance_methods`, `instance_of?`, `is_a?`, `methods`, `tainted?`, `to_json`, `untrusted?`, `||`

**Kernel** (6 methods):
`gauntlet`, `print`, `puts`, `raise`, `require`, `require_relative`

**Set** (3 methods): `[]`, `each`, `initialize`

**Symbol** (2 methods): `==`, `to_s`

**Regexp** (2 methods): `===`, `=~`

**Boolean** (2 methods): `!=`, `==`

**Proc** (1 method): `call`

**MatchData** (1 method): `[]`

### Mixins

**Comparable** (via `include Comparable`):
Requires `<=>`, provides `<`, `>`, `<=`, `>=`, `==`, `between?`, `clamp`

**Enumerable** (via `include Enumerable`):
Requires `each`, provides `all?`, `any?`, `count`, `each_with_index`, `find`, `flat_map`, `include?`, `map`, `max`, `min`, `none?`, `reduce`, `reject`, `select`, `sort`, `sort_by`, `sum`, `to_a`

### Library Facades

Facades map Ruby standard library modules to Go equivalents:

- **Base64** (Tier 1 — pure JSON): `encode64`, `decode64`, `strict_encode64`, `strict_decode64`, `urlsafe_encode64`, `urlsafe_decode64`
- **SecureRandom** (Tier 2 — Go shims): `hex`, `base64`, `urlsafe_base64`, `random_number`, `uuid`, `random_bytes`, `alphanumeric`, `choose`, `rand`, `uniform`
- **Digest** (Tier 2 — Go shims): `Digest::SHA256`, `Digest::SHA384`, `Digest::SHA512` — `.hexdigest`, `.digest`, `.base64digest`
- **JSON** (Tier 2 — Go shims): `JSON.generate`, `JSON.pretty_generate`, `JSON.dump`, `.to_json` (on all types)

---

## Remaining Work

### Language Features
- **Module methods** (`def self.x` inside modules) — type system needs module-level method resolution
- **`def self.x` on modules** as callable namespace functions
- Regexp enhancements: `String#scan` returning captures
- `Array#flatten` with depth argument
- `String#encode` / encoding handling

### Coverage Gaps (from `thanos report`)
The `thanos report` command compares implemented methods against Ruby's MRI. Key gaps:
- **Array**: `append`, `assoc`, `bsearch`, `cycle`, `difference`, `keep_if`, `minmax`, `slice`, `to_h`, `union`
- **Hash**: `assoc`, `compare_by_identity`, `each_pair`, `flatten`, `rassoc`, `to_h`
- **String**: `b`, `center` (improvements), `crypt`, `encode`, `encoding`, `force_encoding`, `slice`, `unpack`
- **Integer**: `abs` (on Integer directly), `ceil`, `floor`, `round`, `succ`, `next`, `to_r`
- **Float**: `abs` (inherited from Numeric), `divmod`, `modulo`, `to_r`, `truncate`

### Quality of Life
- Better error messages with "did you mean?" suggestions
- Source maps / line correlation
- Compiler warnings for unsupported features

---

## What We're Not Doing

Per the project's design philosophy, these are explicitly out of scope:
- Full metaprogramming (`eval`, `class_eval`, `define_method`, `method_missing`)
- Ruby gem/library dependencies (beyond facades)
- Heterogeneous collections
- Dynamic typing / duck typing simulation
- Full Ruby runtime semantics
- `BEGIN`/`END` blocks

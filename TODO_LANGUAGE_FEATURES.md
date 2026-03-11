# TODO: Missing Ruby Language Features in Thanos

This document lists missing Ruby language features ranked from easiest to hardest to implement, based on analysis of the codebase, README, and `thanos report` output.

## **EASIEST (1-3 days)**

### Basic Method Implementations

1. **Array basics** - `Array#first`, `Array#last`, `Array#empty?`, `Array#push`, `Array#pop`
2. **Hash basics** - `Hash#empty?`, `Hash#size`, `Hash#clear`, `Hash#[]`, `Hash#[]=`
3. **String basics** - `String#[]`, `String#length`, `String#empty?`, `String#reverse`
4. **Integer basics** - `Integer#even?`, `Integer#odd?`, `Integer#chr`, `Integer#abs`
5. **Range basics** - `Range#begin`, `Range#end`, `Range#include?`, `Range#cover?`
6. **Symbol basics** - `Symbol#size`, `Symbol#length`, `Symbol#empty?`

### Simple Operators

7. **Unary operators** - `+@`, `-@` for all numeric types
8. **String operators** - `String#+`, `String#*`
9. **Array operators** - `Array#+`, `Array#*`, `Array#&`, `Array#|`
10. **Comparison operators** - `<`, `>`, `<=`, `>=` for String, Symbol, Hash

## **EASY (3-7 days)**

### Collection Enumerables

11. **Basic enumeration** - `all?`, `any?`, `count`, `none?`, `one?` for Array, Hash, Range
12. **Array mutators** - `Array#shift`, `Array#unshift`, `Array#clear`, `Array#delete`
13. **Hash mutators** - `Hash#delete`, `Hash#shift`, `Hash#store`
14. **String mutators** - `String#upcase!`, `String#downcase!`, `String#strip!`, `String#reverse!`

### Core Missing Classes

15. **MatchData methods** - `MatchData#begin`, `MatchData#end`, `MatchData#string`, `MatchData#regexp`
16. **Proc basics** - `Proc#[]`, `Proc#arity`, `Proc#lambda?`

### Language Features from README

17. **`=begin`/`=end` comments** - Multi-line comment support
18. **Alternative interpolation** - `#@foo` syntax (explicitly excluded but easy)

## **MEDIUM (1-2 weeks)**

### Advanced Collections

19. **Array enumeration** - `Array#select`, `Array#reject`, `Array#map`, `Array#each_with_index`
20. **Hash enumeration** - `Hash#select`, `Hash#reject`, `Hash#map`, `Hash#each_pair`
21. **Array slicing** - `Array#slice`, `Array#slice!`, `Array#[]` with ranges
22. **String scanning** - `String#scan`, `String#gsub`, `String#sub`, `String#match`

### Missing Core Classes (README Priority)

23. **Basic Struct class** - Constructor and attribute access
24. **Basic Time class** - Creation, formatting, arithmetic
25. **Basic Date class** - Creation, formatting, arithmetic

### Object Model

26. **Class variables** - `@@var` implementation (parser exists)
27. **Global variables** - `$var` implementation (parser exists)
28. **Method aliasing** - `alias` keyword
29. **`BEGIN` and `END` blocks** - Explicitly excluded but requested

## **HARD (2-4 weeks)**

### Advanced Object Model

30. **Module inclusion** - `include` and `extend`
31. **Method visibility** - Proper `private`/`protected` enforcement
32. **Class methods** - `self.method` and `class << self`
33. **Constants scoping** - Proper `::` resolution

### Advanced Language Features

34. **Multiple assignment** - `a, b = [1, 2]` (parser supports, needs compiler)
35. **Splat operators** - Full `*args` and `**kwargs` support
36. **Keyword arguments** - Named parameters with defaults
37. **Case equality** - `===` operator for all types

### File I/O (Large Missing Set)

38. **File class implementation** - The 259 missing File methods represent a major subsystem

## **VERY HARD (1-2 months)**

### Advanced Metaprogramming

39. **`define_method`** - Dynamic method definition
40. **`method_missing`** - Dynamic dispatch
41. **`send`/`public_send`** - Dynamic invocation
42. **`const_missing`** - Dynamic constants

### Exception Handling

43. **Basic begin/rescue/end** - Exception catching
44. **Advanced exception handling** - `ensure`, `retry`, propagation

### Ruby-ffi Integration (README Long-term Goal)

45. **Automatic wrapper generation** - For C extension compatibility

## **EXTREMELY HARD (2+ months)**

### Major Architectural Changes

46. **Full metaprogramming** - `eval`, `class_eval`, `instance_eval`
47. **Dynamic typing simulation** - Union types for duck typing
48. **Runtime method dispatch** - Full dynamic behavior
49. **Module system rewrite** - Full Ruby module semantics
50. **Heterogeneous collections** - Mixed-type arrays/hashes (README limitation)

### Advanced Runtime Features

51. **Hash ordering preservation** - Overcome Go map limitations (README limitation)
52. **Full dependency system** - Ruby library integration (README says "out of the question")
53. **Complete Ruby runtime** - Full MRI compatibility (antithetical to project goals)

## Implementation Notes

### Quick Wins (Items 1-18)
These represent the best return on investment - basic method implementations that follow existing patterns in the codebase.

### README Priorities
- **Struct**, **Date**, **Time** classes explicitly mentioned as missing
- **File I/O** represents largest single missing feature set (259 methods)
- Exception handling called "one of the largest differences"

### Explicit Limitations
Per README, these are intentionally excluded or extremely difficult:
- Metaprogramming beyond basic level
- Ruby library dependencies
- Heterogeneous arrays/hashes
- Hash ordering guarantees
- Full Ruby runtime semantics

### Architecture Requirements
Items 30+ require significant changes to:
- Type system for dynamic behavior
- Object model for full inheritance
- Runtime for method dispatch
- Memory model for Ruby semantics

## Getting Started

For contributors, start with items 1-10 as they:
1. Follow existing implementation patterns
2. Have clear, testable outcomes
3. Provide immediate value to users
4. Don't require architectural changes

Each implementation should include:
- Method implementation in appropriate `types/*.go` file
- Test cases in `compiler/testdata/`
- Gauntlet test for runtime verification
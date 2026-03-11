# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Thanos is a source-to-source compiler that translates Ruby code into human-readable Go code. It's designed as a porting aid rather than a runtime replacement.

## Build and Development Commands

### Core Commands
- `go build` - Build the thanos binary
- `go run main.go <command>` - Run thanos directly
- `thanos help` - Show available commands after building

### Testing
- `go test ./compiler` - Run style tests (Ruby input → Go output validation)
- `go test ./compiler -filename <name>` - Run a specific style test file
- `thanos test` - Run gauntlet tests (end-to-end Ruby vs Go execution comparison)
- `thanos test --help` - Show test options

### Thanos-Specific Commands
- `thanos compile <ruby_file>` - Compile Ruby source to Go
- `thanos report` - Show missing methods from built-in types
- `thanos exec <ruby_file>` - Execute Ruby through compilation pipeline

## Architecture

### Core Components

**Parser** (`parser/`):
- `ruby.y` - yacc grammar file for Ruby parsing (regenerate with `gen_parser.sh`)
- `lexer.go` - Ruby lexer implementation
- AST node types spread across: `class.go`, `methods.go`, `control_flow.go`, `statements.go`, `node.go`
- `root.go` - Root AST node, analysis orchestration, scope management
- `program.go` - Multi-file support (`require_relative`)

**Type System** (`types/`):
- Type inference engine with `Type` interface
- Predefined types for Ruby primitives and stdlib classes
- `Array`, `String`, `Hash` implementations in respective files
- Method specifications with `TransformAST` functions

**Compiler** (`compiler/`):
- Translates type-annotated Ruby AST to Go AST
- `expr.go` - Expression compilation logic
- `class.go` - Class and method compilation
- Uses `go/ast` package for Go code generation

**Standard Library** (`stdlib/`):
- Go implementations of Ruby methods that need runtime support
- `OrderedMap` for insertion-ordered hash semantics
- Compatibility layers (e.g., `MatchData` for regex operations)
- Generic implementations leveraging Go 1.18+ generics

**Facades** (`facades/`):
- JSON-driven mapping of Ruby stdlib modules to Go equivalents
- Embedded via Go embed, loaded automatically during parsing
- Supports Base64, SecureRandom, Digest, JSON, CSV, Net::HTTP

**Shims** (`shims/`):
- Go glue code for facades that need semantic translation
- Separate from `stdlib/` (which is language runtime support)

### Translation Pipeline

1. **Lexing/Parsing**: Ruby source → AST (`parser.Root`)
2. **Type Inference**: Call `Analyze()` on `*parser.Root`
3. **Compilation**: `compiler.Compile` translates to Go AST
4. **Code Generation**: Go AST → formatted Go source

## Key Design Patterns

### Method Implementation
Built-in Ruby methods are implemented via `MethodSpec` in the `types` package:
- `ReturnType`: Function determining the return type
- `TransformAST`: Function generating Go AST statements and expressions

### AST Utilities
The `bst` package provides AST generation helpers:
- `bst.Call()` - Function/method calls
- `bst.Assign()` - Variable assignments
- `bst.Binary()` - Binary expressions
- `bst.Int()` - Integer literals

### Testing Strategy
Two complementary test approaches:
1. **Style tests**: Validate Go output formatting and structure
2. **Gauntlet tests**: Verify execution equivalence using `gauntlet()` pseudo-method

**IMPORTANT**: Never test compilation by piping Ruby through `thanos compile` via stdin/shell redirection. Always write a gauntlet test (in `tests/*.rb`) or a style test (in `compiler/testdata/ruby/*.rb` with expected output in `compiler/testdata/go/*.go`) and run it with `go run main.go test` or `go test ./compiler`. For throwaway/scratch tests, use the `scratch/` directory (gitignored).

## Important Limitations

- No support for metaprogramming or Ruby runtime features
- Type hints rely on tracking method calls to literal values
- Heterogeneous arrays/hashes not supported

## File Organization

- `main.go` - Entry point, delegates to `cmd.Execute()`
- `cmd/` - Cobra CLI commands (compile, test, report, exec)
- `parser/` - Ruby parsing, AST generation, type inference
- `compiler/` - Ruby-to-Go compilation logic (uses `go/ast`)
- `types/` - Type system, method specs, mixins
- `stdlib/` - Go runtime support for Ruby methods
- `facades/` - JSON facade definitions for Ruby stdlib modules
- `shims/` - Go glue code for library facades
- `tests/` - Gauntlet test Ruby files (458 tests)
- `bst/` - AST building utilities
- `scratch/` - Temporary test files (gitignored)
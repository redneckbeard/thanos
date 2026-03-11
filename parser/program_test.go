package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProgram(t *testing.T) {
	dir := t.TempDir()

	// Write a helper file
	helperPath := filepath.Join(dir, "helper.rb")
	os.WriteFile(helperPath, []byte(`
class Helper
  def initialize(val)
    @val = val
  end

  def value
    @val
  end
end
`), 0644)

	// Write entry file that requires the helper
	entryPath := filepath.Join(dir, "main.rb")
	os.WriteFile(entryPath, []byte(`
require_relative 'helper'

h = Helper.new(42)
puts h.value
`), 0644)

	root, err := ParseProgram(entryPath)
	if err != nil {
		t.Fatalf("ParseProgram failed: %v", err)
	}

	// Should have the Helper class from the required file
	if len(root.Classes) != 1 {
		t.Fatalf("expected 1 class, got %d", len(root.Classes))
	}
	if root.Classes[0].Name() != "Helper" {
		t.Fatalf("expected class Helper, got %s", root.Classes[0].Name())
	}

	// require_relative should be stripped from statements
	for _, stmt := range root.Statements {
		if call, ok := stmt.(*MethodCall); ok && call.MethodName == "require_relative" {
			t.Fatal("require_relative should have been stripped from statements")
		}
	}
}

func TestParseProgramChained(t *testing.T) {
	dir := t.TempDir()

	// A -> B -> C
	os.WriteFile(filepath.Join(dir, "base.rb"), []byte(`
class Base
  def hello
    "hi"
  end
end
`), 0644)

	os.WriteFile(filepath.Join(dir, "mid.rb"), []byte(`
require_relative 'base'

class Mid < Base
  def greet
    hello
  end
end
`), 0644)

	os.WriteFile(filepath.Join(dir, "entry.rb"), []byte(`
require_relative 'mid'

m = Mid.new
puts m.greet
`), 0644)

	root, err := ParseProgram(filepath.Join(dir, "entry.rb"))
	if err != nil {
		t.Fatalf("ParseProgram failed: %v", err)
	}

	if len(root.Classes) != 2 {
		t.Fatalf("expected 2 classes, got %d", len(root.Classes))
	}
}

func TestParseProgramDedup(t *testing.T) {
	dir := t.TempDir()

	// Both B and C require A; entry requires B and C.
	// A should only be loaded once.
	os.WriteFile(filepath.Join(dir, "shared.rb"), []byte(`
class Shared
  def val
    1
  end
end
`), 0644)

	os.WriteFile(filepath.Join(dir, "b.rb"), []byte(`
require_relative 'shared'
`), 0644)

	os.WriteFile(filepath.Join(dir, "c.rb"), []byte(`
require_relative 'shared'
`), 0644)

	os.WriteFile(filepath.Join(dir, "entry.rb"), []byte(`
require_relative 'b'
require_relative 'c'

s = Shared.new
puts s.val
`), 0644)

	root, err := ParseProgram(filepath.Join(dir, "entry.rb"))
	if err != nil {
		t.Fatalf("ParseProgram failed: %v", err)
	}

	// Shared should appear only once
	if len(root.Classes) != 1 {
		t.Fatalf("expected 1 class (deduped), got %d", len(root.Classes))
	}
}

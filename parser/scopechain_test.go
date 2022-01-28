package parser

import (
	"testing"
)

func TestScopeChain(t *testing.T) {
	ruby := `
		def foo; return "foo"; end
		def bar; return "bar"; end
		def baz; return "baz"; end
		def quux; return "quux"; end

		foo()
		bar()
		baz()
		quux()
		`
	scopeNames := [][]string{
		{"__main__", "foo"},
		{"__main__", "bar"},
		{"__main__", "baz"},
		{"__main__", "quux"},
	}
	p, err := ParseString(ruby)
	if err != nil {
		t.Fatal(err)
	}
	for _, scopeName := range scopeNames {
		method := p.MethodSetStack.Peek().Methods[scopeName[1]]
		for j, scope := range scopeName {
			if method.Scope[j].Name() != scope {
				t.Errorf("expected scope name to be '%s' but found '%s'", scope, method.Scope[j].Name())
			}
		}
	}
}

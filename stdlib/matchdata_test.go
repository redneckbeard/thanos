package stdlib

import (
	"regexp"
	"testing"
)

func TestMatchData(t *testing.T) {
	patt1 := regexp.MustCompile("foo")
	patt2 := regexp.MustCompile("f(oo)")
	patt3 := regexp.MustCompile("f(oo)(?P<bar>bar)?")
	md := NewMatchData(patt1, "bar")
	if md != nil {
		t.Errorf(`/foo/ should not have matched "bar", should have returned nil`)
	}
	md = NewMatchData(patt1, "foo")
	if md.Get(0) != "foo" {
		t.Errorf(`expected /foo/ to match "foo", didn't have complete match present`)
	}
	if md.Get(1) != "" {
		t.Errorf(`expected first submatch to be zero value`)
	}
	md = NewMatchData(patt2, "foo")
	if md.Get(1) != "oo" {
		t.Errorf(`expected first submatch to be "oo"`)
	}
	if md.GetByName("key") != "" {
		t.Errorf(`expected submatch retrieval by name to fail when capture is unnamed`)
	}
	md = NewMatchData(patt3, "foobar")
	if md.Get(1) != "oo" {
		t.Errorf(`expected first submatch to be "oo"`)
	}
	if md.Get(2) != "bar" {
		t.Errorf(`expected second submatch to be "bar"`)
	}
	if md.GetByName("bar") != "bar" {
		t.Errorf(`expected submatch retrieval by name to work`)
	}
}

func TestConvertFromGsub(t *testing.T) {
	tests := []struct{ regex, ruby, expected string }{
		{
			`[aeiou]`,
			"*",
			"*",
		},
		{
			`([aeiou])`,
			`<\1>`,
			"<${1}>",
		},
		{
			`(?P<foo>[aeiou])`,
			`{\k<foo>}`,
			"{${1}}",
		},
	}

	for _, tt := range tests {
		patt := regexp.MustCompile(tt.regex)
		converted := ConvertFromGsub(patt, tt.ruby)
		if converted != tt.expected {
			t.Fatalf("Expected '%s' to convert to '%s' but got '%s'", tt.ruby, tt.expected, converted)
		}
	}
}

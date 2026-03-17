package stdlib

import "reflect"

// Metaclass represents a Ruby class object at runtime. In Ruby, classes are
// first-class objects with methods like .name, .new, and == comparison. This
// type reifies that concept in Go so that expressions like
// `self.class == other.class` compile naturally.
//
// Each compiled class gets a package-level Metaclass value:
//
//	var ChangeClass = stdlib.NewMetaclass[Change]("Diff::LCS::Change")
//
// Metaclass is a value type (not a pointer) so that == comparison works
// directly via Go's built-in equality on comparable structs.
type Metaclass struct {
	RubyName string
	GoType   reflect.Type
}

// NewMetaclass creates a Metaclass for Go type T with the specified Ruby name.
func NewMetaclass[T any](rubyName string) Metaclass {
	return Metaclass{
		RubyName: rubyName,
		GoType:   reflect.TypeOf((*T)(nil)).Elem(),
	}
}

// Name returns the Ruby class name (e.g., "Diff::LCS::Change").
func (m Metaclass) Name() string {
	return m.RubyName
}

package types

import (
	"testing"
)

func TestClassRegistry(t *testing.T) {
	t.Skip()
	registry := &classRegistry{registry: make(map[string]*Class)}

	newProto("Foo", "Bar", registry)

	class, err := registry.Get("Foo")

	if err == nil || class != nil {
		t.Fatal("Expected ClassRegistry.Get to error without initializing first")
	}
	err = registry.Initialize()

	expected := "Class 'Foo' described as having parent 'Bar' but no class 'Bar' was ever registered"

	if err == nil || err.Error() != expected {
		t.Fatalf(`Expected error "%s" but got "%s"`, expected, err)
	}

	newProto("Bar", "Baz", registry)
	newProto("Baz", "", registry)

	err = registry.Initialize()

	if err != nil {
		t.Fatalf(`Encountered unexpected error: "%s"`, err)
	}
}

func TestMethodResolution(t *testing.T) {
	t.Skip()
	registry := &classRegistry{registry: make(map[string]*Class)}

	foo := newProto("Foo", "Bar", registry)
	bar := newProto("Bar", "", registry)
	bar.Def("method", MethodSpec{})
	registry.Initialize()
	if !foo.HasMethod("method", false) {
		t.Fatal("'Foo' failed to inherit method 'method'")
	}
}

func TestClassNewDelegatesToInstanceInitialize(t *testing.T) {
	t.Skip()
	registry := &classRegistry{registry: make(map[string]*Class)}
	foo := newProto("Foo", "", registry)
	registry.Initialize()
	fakeType := Simple(100)
	foo.Def("initialize", MethodSpec{
		ReturnType: func(receiverType Type, blockReturnType Type, args []Type) (Type, error) {
			return fakeType, nil
		},
	})
	class, _ := registry.Get("Foo")
	if !class.HasMethod("new") {
		t.Fatal("`Foo#new` not handled")
	}
	spec := class.MustResolve("new")
	if typ, _ := spec.ReturnType(nil, nil, []Type{}); typ != fakeType {
		t.Fatal("Not mapping #new to instance #initialize")
	}
}

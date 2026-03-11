package types

var ExceptionClasses = []string{
	"StandardError",
	"RuntimeError",
	"ArgumentError",
	"TypeError",
	"ZeroDivisionError",
	"NameError",
	"IndexError",
	"KeyError",
	"RangeError",
	"IOError",
	"NotImplementedError",
	"StopIteration",
	"RegexpError",
}

// ExceptionParents maps each exception class to its parent for inheritance matching
var ExceptionParents = map[string]string{
	"RuntimeError":        "StandardError",
	"ArgumentError":       "StandardError",
	"TypeError":           "StandardError",
	"ZeroDivisionError":   "StandardError",
	"NameError":           "StandardError",
	"IndexError":          "StandardError",
	"KeyError":            "StandardError",
	"RangeError":          "StandardError",
	"IOError":             "StandardError",
	"NotImplementedError": "StandardError",
	"StopIteration":       "StandardError",
	"RegexpError":         "StandardError",
}

func init() {
	for _, name := range ExceptionClasses {
		parent := "Object"
		if p, ok := ExceptionParents[name]; ok {
			parent = p
		}
		NewClass(name, parent, nil, ClassRegistry)
	}
}

// IsExceptionClass returns true if the given name is a registered exception class
func IsExceptionClass(name string) bool {
	for _, cls := range ExceptionClasses {
		if cls == name {
			return true
		}
	}
	return name == "StandardError"
}

// IsAncestorException returns true if ancestor is an ancestor of child in the exception hierarchy
func IsAncestorException(child, ancestor string) bool {
	for child != "" {
		if child == ancestor {
			return true
		}
		child = ExceptionParents[child]
	}
	return false
}

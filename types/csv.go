package types

// CSVClass is the empty class shell for CSV. Method specs are populated by
// csv/types.go init() — this avoids an import cycle while keeping the class
// in the ClassRegistry for ConstantNode resolution.
var CSVClass = NewClass("CSV", "Object", nil, ClassRegistry)

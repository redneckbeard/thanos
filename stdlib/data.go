package stdlib

import (
	"fmt"
	"strings"
)

// DataInspect formats a Data.define instance as "#<data ClassName field1=val1, field2=val2>".
func DataInspect(className string, fieldNames []string, fieldValues []interface{}) string {
	pairs := make([]string, len(fieldNames))
	for i, name := range fieldNames {
		pairs[i] = fmt.Sprintf("%s=%v", name, fieldValues[i])
	}
	return fmt.Sprintf("#<data %s %s>", className, strings.Join(pairs, ", "))
}

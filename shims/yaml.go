package shims

import (
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLLoad parses a YAML string and returns the result as a Go interface{}.
// For simple scalar values, returns string. For complex structures, returns
// map[string]interface{} or []interface{}.
func YAMLLoad(s string) interface{} {
	var result interface{}
	yaml.Unmarshal([]byte(s), &result)
	return result
}

// YAMLDump serializes a Go value to a YAML string.
// Prepends "--- " to match Ruby's YAML.dump output format.
func YAMLDump(v interface{}) string {
	out, _ := yaml.Marshal(v)
	return "--- " + string(out)
}

// YAMLLoadFile reads and parses a YAML file.
func YAMLLoadFile(filename string) interface{} {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	var result interface{}
	yaml.Unmarshal(data, &result)
	return result
}

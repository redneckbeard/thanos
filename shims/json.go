package shims

import (
	"encoding/json"
	"fmt"
)

// JSONGenerate converts any value to a JSON string.
// Mirrors Ruby's JSON.generate(obj) and obj.to_json.
func JSONGenerate(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// JSONPrettyGenerate converts any value to a pretty-printed JSON string.
// Mirrors Ruby's JSON.pretty_generate(obj).
func JSONPrettyGenerate(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// JSONParse parses a JSON string into a map[string]string.
// Mirrors Ruby's JSON.parse(str) for object inputs.
// Values are coerced to strings since thanos hashes are homogeneously typed.
func JSONParse(s string) map[string]string {
	var raw map[string]interface{}
	json.Unmarshal([]byte(s), &raw)
	result := make(map[string]string, len(raw))
	for k, v := range raw {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

package shims

import "encoding/json"

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

// JSONParse parses a JSON string into a map[string]interface{}.
// Mirrors Ruby's JSON.parse(str) for object inputs.
func JSONParse(s string) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal([]byte(s), &result)
	return result
}

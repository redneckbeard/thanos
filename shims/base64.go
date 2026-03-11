package shims

// AppendNewline adds a trailing newline, matching Ruby's Base64.encode64 behavior.
func AppendNewline(s string) string {
	return s + "\n"
}

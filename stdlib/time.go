package stdlib

import "strings"

// RubyStrftime converts a Ruby strftime format string to a Go time.Format layout string.
// Go uses the reference time Mon Jan 2 15:04:05 MST 2006 (01/02 03:04:05PM '06 -0700).
func RubyStrftime(rubyFmt string) string {
	replacements := []struct {
		ruby, golang string
	}{
		// Order matters: longer sequences first to avoid partial matches
		{"%-d", "2"},    // day without leading zero
		{"%-m", "1"},    // month without leading zero
		{"%-H", "15"},   // 24-hour without leading zero (Go doesn't distinguish)
		{"%-I", "3"},    // 12-hour without leading zero
		{"%-M", "4"},    // minute without leading zero
		{"%-S", "5"},    // second without leading zero
		{"%Y", "2006"},  // 4-digit year
		{"%C", "20"},    // century (first two digits of year)
		{"%y", "06"},    // 2-digit year
		{"%m", "01"},    // month with leading zero
		{"%d", "02"},    // day with leading zero
		{"%H", "15"},    // 24-hour
		{"%I", "03"},    // 12-hour with leading zero
		{"%M", "04"},    // minute
		{"%S", "05"},    // second
		{"%L", ".000"},  // milliseconds
		{"%N", ".000000000"}, // nanoseconds
		{"%p", "PM"},    // AM/PM
		{"%P", "pm"},    // am/pm
		{"%Z", "MST"},   // timezone abbreviation
		{"%z", "-0700"}, // timezone offset
		{"%:z", "-07:00"}, // timezone offset with colon
		{"%A", "Monday"},    // full weekday name
		{"%a", "Mon"},       // abbreviated weekday name
		{"%B", "January"},   // full month name
		{"%b", "Jan"},       // abbreviated month name
		{"%h", "Jan"},       // same as %b
		{"%R", "15:04"},     // 24-hour HH:MM
		{"%T", "15:04:05"},  // 24-hour HH:MM:SS
		{"%r", "03:04:05 PM"}, // 12-hour with AM/PM
		{"%D", "01/02/06"},  // date as mm/dd/yy
		{"%F", "2006-01-02"}, // ISO 8601 date
		{"%n", "\n"},        // newline
		{"%t", "\t"},        // tab
		{"%%", "%"},         // literal percent
	}

	result := rubyFmt
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.ruby, r.golang)
	}
	return result
}

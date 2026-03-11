package csv

import (
	"encoding/csv"
	"os"
	"strings"
)

// Read reads an entire CSV file and returns all records as [][]string.
func Read(filename string) [][]string {
	f, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer f.Close()
	records, _ := csv.NewReader(f).ReadAll()
	return records
}

// ReadWithOptions reads a CSV file with a custom column separator.
func ReadWithOptions(filename string, colSep rune) [][]string {
	f, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = colSep
	records, _ := r.ReadAll()
	return records
}

// Parse parses a CSV string and returns all records as [][]string.
func Parse(s string) [][]string {
	records, _ := csv.NewReader(strings.NewReader(s)).ReadAll()
	return records
}

// ParseWithOptions parses a CSV string with a custom column separator.
func ParseWithOptions(s string, colSep rune) [][]string {
	r := csv.NewReader(strings.NewReader(s))
	r.Comma = colSep
	records, _ := r.ReadAll()
	return records
}

// ReadWithHeaders reads a CSV file where the first row is headers,
// returning a Table of named Rows.
func ReadWithHeaders(filename string) *Table {
	return NewTable(Read(filename))
}

// ReadWithHeadersAndOptions reads a CSV file with headers and custom separator.
func ReadWithHeadersAndOptions(filename string, colSep rune) *Table {
	return NewTable(ReadWithOptions(filename, colSep))
}

// ParseWithHeaders parses a CSV string where the first row is headers.
func ParseWithHeaders(s string) *Table {
	return NewTable(Parse(s))
}

// ParseWithHeadersAndOptions parses a CSV string with headers and custom separator.
func ParseWithHeadersAndOptions(s string, colSep rune) *Table {
	return NewTable(ParseWithOptions(s, colSep))
}

// GenerateLine formats a single row as a CSV line string.
func GenerateLine(row []string) string {
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Write(row)
	w.Flush()
	return buf.String()
}

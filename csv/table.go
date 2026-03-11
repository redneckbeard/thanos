package csv

// Table represents a CSV file parsed with headers (Ruby's CSV::Table).
type Table struct {
	headers []string
	rows    []*Row
}

// NewTable creates a Table from a slice of records where the first row
// contains headers and subsequent rows contain data.
func NewTable(records [][]string) *Table {
	if len(records) == 0 {
		return &Table{}
	}
	headers := records[0]
	rows := make([]*Row, 0, len(records)-1)
	for _, rec := range records[1:] {
		rows = append(rows, NewRow(headers, rec))
	}
	return &Table{headers: headers, rows: rows}
}

// Get returns the row at index i.
func (t *Table) Get(i int) *Row {
	if i < len(t.rows) {
		return t.rows[i]
	}
	return nil
}

// Headers returns the column names.
func (t *Table) Headers() []string {
	return t.headers
}

// Len returns the number of data rows (excludes the header row).
func (t *Table) Len() int {
	return len(t.rows)
}

// ToSlice returns all data rows as a slice of *Row.
func (t *Table) ToSlice() []*Row {
	return t.rows
}

// ToArray returns all rows including the header row as [][]string,
// matching Ruby's CSV::Table#to_a behavior.
func (t *Table) ToArray() [][]string {
	result := make([][]string, 0, len(t.rows)+1)
	result = append(result, t.headers)
	for _, row := range t.rows {
		result = append(result, row.fields)
	}
	return result
}

// ToCsv returns the table as a CSV-formatted string including headers.
func (t *Table) ToCsv() string {
	var s string
	s += GenerateLine(t.headers)
	for _, row := range t.rows {
		s += GenerateLine(row.fields)
	}
	return s
}

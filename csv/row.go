package csv

// Row represents a single CSV row with named column access (Ruby's CSV::Row).
type Row struct {
	headers []string
	fields  []string
	index   map[string]int
}

// NewRow creates a Row from parallel header and field slices.
func NewRow(headers, fields []string) *Row {
	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		idx[h] = i
	}
	return &Row{headers: headers, fields: fields, index: idx}
}

// Get returns the field value for the given column name.
// Returns "" if the column doesn't exist.
func (r *Row) Get(col string) string {
	if i, ok := r.index[col]; ok && i < len(r.fields) {
		return r.fields[i]
	}
	return ""
}

// Headers returns the column names.
func (r *Row) Headers() []string {
	return r.headers
}

// Fields returns the field values.
func (r *Row) Fields() []string {
	return r.fields
}

// ToHash returns a map of column names to field values.
func (r *Row) ToHash() map[string]string {
	m := make(map[string]string, len(r.headers))
	for i, h := range r.headers {
		if i < len(r.fields) {
			m[h] = r.fields[i]
		}
	}
	return m
}

// Set sets the field value for the given column name.
// If the column doesn't exist, it appends a new header+field.
func (r *Row) Set(col, val string) {
	if i, ok := r.index[col]; ok && i < len(r.fields) {
		r.fields[i] = val
		return
	}
	r.index[col] = len(r.headers)
	r.headers = append(r.headers, col)
	r.fields = append(r.fields, val)
}

// Delete removes a field by header name and returns [header, value].
// Returns nil if the column doesn't exist.
func (r *Row) Delete(col string) []string {
	i, ok := r.index[col]
	if !ok {
		return nil
	}
	val := ""
	if i < len(r.fields) {
		val = r.fields[i]
		r.fields = append(r.fields[:i], r.fields[i+1:]...)
	}
	r.headers = append(r.headers[:i], r.headers[i+1:]...)
	delete(r.index, col)
	// Rebuild index
	for j := i; j < len(r.headers); j++ {
		r.index[r.headers[j]] = j
	}
	return []string{col, val}
}

// ToCsv returns the row as a CSV-formatted string line.
func (r *Row) ToCsv() string {
	return GenerateLine(r.fields)
}

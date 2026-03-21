# Generated Go output for showcase.rb

This is the Go code thanos produces from [`showcase.rb`](showcase.rb). The diff-lcs gem compiles to a separate package under `diff/lcs/` (not shown here).

```go
package main

import (
	"fmt"
	"strconv"
	"strings"
	"tmpmod/diff/lcs"

	"github.com/redneckbeard/thanos/csv"
	"github.com/redneckbeard/thanos/net_http"
	"github.com/redneckbeard/thanos/shims"
	"github.com/redneckbeard/thanos/stdlib"
)

func Fetch_csv(host, path string) string {
	body := ""
	http := net_http.NewClient(host, 443, true)
	response := http.Get(path)
	body = response.Body()
	return body
}

func Csv_to_lines(text string) []string {
	table := csv.ParseWithHeaders(text)
	lines := []string{}
	for _, row := range table.ToSlice() {
		lines = append(lines, strings.Join(row.Fields(), ","))
	}
	return lines
}

func main() {
	base := "/redneckbeard/thanos/main/examples"
	host := "raw.githubusercontent.com"
	fmt.Println("Fetching CSVs from GitHub...")
	v1_text := Fetch_csv(host, base+"/students_v1.csv")
	v2_text := Fetch_csv(host, base+"/students_v2.csv")
	fmt.Println("Parsing CSV data...")
	v1_lines := Csv_to_lines(v1_text)
	v2_lines := Csv_to_lines(v2_text)
	fmt.Println("v1: " + strconv.Itoa(len(v1_lines)) + " rows")
	fmt.Println("v2: " + strconv.Itoa(len(v2_lines)) + " rows")
	fmt.Println("")
	fmt.Println("Running diff...")
	common := lcs.Lcs(v1_lines, v2_lines, nil)
	matching := len(common)
	total := len(v1_lines)
	mismatched := total - matching
	om := stdlib.NewOrderedMap[interface{}, interface{}]()
	diffs := om
	i := 0
	for i < total {
		if v1_lines[i] != v2_lines[i] {
			diffs.Set(strconv.Itoa(i+1), v1_lines[i]+" -> "+v2_lines[i])
		}
		i++
	}
	om1 := stdlib.NewOrderedMap[string, string]()
	om1.Set("total_lines", strconv.Itoa(total))
	om1.Set("matching_lines", strconv.Itoa(matching))
	om1.Set("mismatched_lines", strconv.Itoa(mismatched))
	om1.Set("diffs_by_line", shims.JSONGenerate(diffs))
	report := om1
	fmt.Println("")
	fmt.Println("=== Diff Report ===")
	fmt.Println(shims.JSONGenerate(report))
}
```

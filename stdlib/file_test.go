package stdlib

import (
	"bufio"
	"os"
	"testing"
)

func TestMakeSplitFunc(t *testing.T) {
	linesFile, _ := os.CreateTemp("", "lines.test")
	defer os.Remove(linesFile.Name())
	linesFile.WriteString(`here are several
lines that end with
newlines but that all
seem to have "that" in them`)

	tests := []struct {
		separator string
		chomp     bool
		lines     []string
	}{
		{
			separator: "\n",
			chomp:     false,
			lines: []string{
				"here are several\n",
				"lines that end with\n",
				"newlines but that all\n",
				"seem to have \"that\" in them",
			},
		},
		{
			separator: "\n",
			chomp:     true,
			lines: []string{
				"here are several",
				"lines that end with",
				"newlines but that all",
				"seem to have \"that\" in them",
			},
		},
		{
			separator: "that",
			chomp:     false,
			lines: []string{
				"here are several\nlines that",
				" end with\nnewlines but that",
				" all\nseem to have \"that",
				"\" in them",
			},
		},
		{
			separator: "that",
			chomp:     true,
			lines: []string{
				"here are several\nlines ",
				" end with\nnewlines but ",
				" all\nseem to have \"",
				"\" in them",
			},
		},
	}
	for _, tt := range tests {
		newlineSplitFile, _ := os.Open(linesFile.Name())
		scanner := bufio.NewScanner(newlineSplitFile)
		scanner.Split(MakeSplitFunc(tt.separator, tt.chomp))
		lineNo := 0
		for scanner.Scan() {
			line := scanner.Text()
			if tt.lines[lineNo] != line {
				t.Fatalf(`Line No. %d did not match for sep "%s" and chomp: %t -- "%s" != "%s"`, lineNo, tt.separator, tt.chomp, line, tt.lines[lineNo])
			}
			lineNo++
		}
	}
}

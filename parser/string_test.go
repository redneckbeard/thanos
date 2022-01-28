package parser

import "testing"

func TestEscapeTranslation(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{`"\a"`, `"\a"`},
		{`"\b"`, `"\b"`},
		{`"\f"`, `"\f"`},
		{`"\n"`, `"\n"`},
		{`"\r"`, `"\r"`},
		{`"\t"`, `"\t"`},
		{`"\v"`, `"\v"`},
		{`"\d\g\h"`, `"dgh"`},
		{`'\d\g\h'`, "`\\d\\g\\h`"},
		{`"\""`, `"\""`},
		{`"\'"`, `"'"`},
		{`'\\'`, "`\\`"},
		{`'\''`, "`'`"},
		{`"\\\""`, `"\\\""`},
		{`%w|x\||`, "`x|`"},
	}
	for i, tt := range tests {
		if caseNum == 0 || caseNum == i+1 {
			p, err := ParseString(tt.in)
			node := p.Statements[0].(*StringNode)
			if tt.out != node.GoString() {
				t.Errorf("[%d] Expected %s but got %s", i+1, tt.out, node.GoString())
				if err != nil {
					t.Errorf("[%d] Parse errors: %s", i+1, err)
				}
			}
		}
	}
}

func TestInvalidEscapes(t *testing.T) {
	tests := []struct {
		in, msg string
	}{
		{`"\e"`, `line 1: \e is not a valid escape sequence in Go strings`},
		{`"\s"`, `line 1: \s is not a valid escape sequence in Go strings`},
		{`"\M"`, `line 1: \M-x, \M-\C-x, and \M-\cx are not valid escape sequences in Go strings`},
		{`"\c"`, `line 1: \c\M-x, \c?, and \C? are not valid escape sequences in Go strings`},
	}
	for i, tt := range tests {
		if caseNum == 0 || caseNum == i+1 {
			_, err := ParseString(tt.in)
			if err == nil {
				t.Errorf("[%d] Expected error '%s' but got none", i+1, tt.msg)
			} else if tt.msg != err.Error() {
				t.Errorf("[%d] Expected error '%s' but got '%s'", i+1, tt.msg, err.Error())
			}
		}
	}
}

package compiler

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/redneckbeard/thanos/parser"
)

func CompareThanosToMRI(program, label string) (string, string, error) {
	cmd := exec.Command("ruby")
	cmd.Stdin = strings.NewReader(program)
	var out, rubyErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &rubyErr
	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf(`Failed to execute Ruby script:
%s

Error: %s`, program, rubyErr.String())
	}
	rubyTmp, _ := os.CreateTemp("", "ruby.results")
	defer os.Remove(rubyTmp.Name())
	rubyTmp.Write(out.Bytes())

	prog, err := parser.ParseString(program)
	if err != nil {
		return "", "", fmt.Errorf("Error parsing '"+program+"': ", err)
	}
	translated, err := Compile(prog)
	goTmp, _ := os.Create("tmp.go")
	defer os.Remove(goTmp.Name())
	goTmp.WriteString(translated)

	cmd = exec.Command("go", "run", goTmp.Name())
	out.Reset()
	cmd.Stdout = &out
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	err = cmd.Run()
	if err != nil {
		return "", translated, fmt.Errorf(`%s
Go compilation of translated source failed for '%s'. Translation:
------
%s
------`, errBuf.String(), label, translated)
	}
	goOutTmp, _ := os.CreateTemp("", "go.results")
	defer os.Remove(goOutTmp.Name())
	goOutTmp.Write(out.Bytes())

	comm := exec.Command("comm", "-23", rubyTmp.Name(), goOutTmp.Name())
	out.Reset()
	errBuf.Reset()
	comm.Stdout = &out
	comm.Stderr = &errBuf
	err = comm.Run()
	if err != nil {
		return "", translated, fmt.Errorf("%s: %s", err, errBuf.String())
	}
	return strings.TrimSpace(out.String()), translated, nil
}

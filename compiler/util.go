package compiler

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/redneckbeard/thanos/parser"
)

func rubyPath() string {
	// THANOS_RUBY env var takes priority
	if p := os.Getenv("THANOS_RUBY"); p != "" {
		return p
	}
	// Try rbenv shim
	home, _ := os.UserHomeDir()
	rbenvRuby := filepath.Join(home, ".rbenv", "shims", "ruby")
	if _, err := os.Stat(rbenvRuby); err == nil {
		return rbenvRuby
	}
	// Fall back to PATH
	return "ruby"
}

func CompareThanosToMRI(program, label string) (string, string, error) {
	cmd := exec.Command(rubyPath())
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
	result, err := Compile(prog)
	mainSrc := result.MainFile()

	// If there are multiple files (module packages), write a temp project directory
	if len(result.Files) > 1 {
		return compareThanosMultiFile(result, rubyTmp.Name(), label)
	}

	// Single file — use the simple path
	goTmp, _ := os.Create("tmp.go")
	defer os.Remove(goTmp.Name())
	goTmp.WriteString(mainSrc)

	cmd = exec.Command("go", "run", goTmp.Name())
	out.Reset()
	cmd.Stdout = &out
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	err = cmd.Run()
	if err != nil {
		return "", mainSrc, fmt.Errorf(`%s
Go compilation of translated source failed for '%s'. Translation:
------
%s
------`, errBuf.String(), label, mainSrc)
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
		return "", mainSrc, fmt.Errorf("%s: %s", err, errBuf.String())
	}
	return strings.TrimSpace(out.String()), mainSrc, nil
}

func compareThanosMultiFile(result *CompileResult, rubyResultsPath, label string) (string, string, error) {
	// Create a temp directory for the multi-file Go project
	tmpDir, err := os.MkdirTemp("", "thanos-test-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Find the thanos stdlib path for the replace directive
	thanosRoot, _ := filepath.Abs(".")
	// If running from a subdirectory (e.g., compiler/), go up
	if _, err := os.Stat(filepath.Join(thanosRoot, "stdlib")); err != nil {
		thanosRoot = filepath.Dir(thanosRoot)
	}
	stdlibPath := filepath.Join(thanosRoot, "stdlib")
	shimsPath := filepath.Join(thanosRoot, "shims")

	// Write go.mod
	goMod := fmt.Sprintf(`module tmpmod

go 1.23

require (
	github.com/redneckbeard/thanos/stdlib v0.0.0
	github.com/redneckbeard/thanos/shims v0.0.0
)

replace github.com/redneckbeard/thanos/stdlib => %s
replace github.com/redneckbeard/thanos/shims => %s
`, stdlibPath, shimsPath)
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)

	// Write all source files
	allSrc := ""
	for path, src := range result.Files {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(src), 0644)
		allSrc += fmt.Sprintf("// === %s ===\n%s\n", path, src)
	}

	// Run go mod tidy to resolve dependencies
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = tmpDir
	var tidyErr bytes.Buffer
	tidy.Stderr = &tidyErr
	if err := tidy.Run(); err != nil {
		return "", allSrc, fmt.Errorf("go mod tidy failed for '%s': %s\nFiles:\n%s", label, tidyErr.String(), allSrc)
	}

	// Run the project
	run := exec.Command("go", "run", ".")
	run.Dir = tmpDir
	var out, errBuf bytes.Buffer
	run.Stdout = &out
	run.Stderr = &errBuf
	err = run.Run()
	if err != nil {
		return "", allSrc, fmt.Errorf(`%s
Go compilation of translated source failed for '%s'. Translation:
------
%s
------`, errBuf.String(), label, allSrc)
	}

	goOutTmp, _ := os.CreateTemp("", "go.results")
	defer os.Remove(goOutTmp.Name())
	goOutTmp.Write(out.Bytes())

	comm := exec.Command("comm", "-23", rubyResultsPath, goOutTmp.Name())
	out.Reset()
	errBuf.Reset()
	comm.Stdout = &out
	comm.Stderr = &errBuf
	err = comm.Run()
	if err != nil {
		return "", allSrc, fmt.Errorf("%s: %s", err, errBuf.String())
	}
	return strings.TrimSpace(out.String()), allSrc, nil
}

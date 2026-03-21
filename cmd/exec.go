/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/redneckbeard/thanos/compiler"
	"github.com/redneckbeard/thanos/parser"
	"github.com/spf13/cobra"
)

var File string

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Compiles and executes the input",
	Long: `'thanos exec' compiles the provided Ruby and immediately executes the Go
	output. Useful for exploring edge cases that might be missing from the test
	suite.`,
	Run: func(cmd *cobra.Command, args []string) {
		if File == "" {
			color.Green("Input your Ruby and execute with Ctrl-D.")
		}
		program, err := parser.ParseFile(File)
		if err != nil {
			fmt.Println(err)
			return
		}
		result, err := compiler.Compile(program)
		if err != nil {
			fmt.Println(err)
			if result != nil {
				fmt.Println(result.MainFile())
			}
			return
		}

		stdout, stderr, err := execCompileResult(result)
		if err != nil {
			allSrc := ""
			for path, src := range result.Files {
				allSrc += fmt.Sprintf("// === %s ===\n%s\n", path, src)
			}
			color.Red("Execution failed for compiled Go:\n------\n%s------\nError: %s", allSrc, stderr)
		} else {
			color.Green(strings.Repeat("-", 20))
			fmt.Print(stdout)
		}
	},
}

// execCompileResult runs a compiled Go project. For single-file results with
// no external dependencies it uses a simple `go run tmp.go`. For multi-file
// results or results that import stdlib/shims, it creates a temporary directory
// with a go.mod and runs `go run .`.
func execCompileResult(result *compiler.CompileResult) (string, string, error) {
	mainSrc := result.MainFile()
	needsModule := len(result.Files) > 1 ||
		strings.Contains(mainSrc, "github.com/redneckbeard/thanos/stdlib") ||
		strings.Contains(mainSrc, "github.com/redneckbeard/thanos/shims")

	if !needsModule {
		return execSingleFile(mainSrc)
	}
	return execMultiFile(result)
}

func execSingleFile(compiled string) (string, string, error) {
	goTmp, _ := os.CreateTemp("", "thanos-exec-*.go")
	defer os.Remove(goTmp.Name())
	goTmp.WriteString(compiled)
	goTmp.Close()

	run := exec.Command("go", "run", goTmp.Name())
	var stderr bytes.Buffer
	run.Stderr = &stderr
	stdout, err := run.Output()
	return string(stdout), stderr.String(), err
}

func execMultiFile(result *compiler.CompileResult) (string, string, error) {
	tmpDir, err := os.MkdirTemp("", "thanos-exec-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Find the thanos root for replace directives
	thanosRoot := findThanosRoot()

	goMod := fmt.Sprintf("module tmpmod\n\ngo 1.23\n\nrequire (\n\tgithub.com/redneckbeard/thanos v0.0.0\n\tgithub.com/redneckbeard/thanos/stdlib v0.0.0\n\tgithub.com/redneckbeard/thanos/shims v0.0.0\n)\n\nreplace github.com/redneckbeard/thanos => %s\nreplace github.com/redneckbeard/thanos/stdlib => %s\nreplace github.com/redneckbeard/thanos/shims => %s\n",
		thanosRoot,
		filepath.Join(thanosRoot, "stdlib"),
		filepath.Join(thanosRoot, "shims"),
	)
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)

	// Write all source files
	for path, src := range result.Files {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(src), 0644)
	}

	// Resolve dependencies
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = tmpDir
	var tidyErr bytes.Buffer
	tidy.Stderr = &tidyErr
	if err := tidy.Run(); err != nil {
		return "", tidyErr.String(), fmt.Errorf("go mod tidy failed: %s", tidyErr.String())
	}

	// Run the project
	run := exec.Command("go", "run", ".")
	run.Dir = tmpDir
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err = run.Run()
	return stdout.String(), stderr.String(), err
}

// findThanosRoot locates the thanos project root by looking for the stdlib/
// directory, walking up from the current executable or working directory.
func findThanosRoot() string {
	// Try the executable's directory first (works when `thanos` is built in-tree)
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 5; i++ {
			if _, err := os.Stat(filepath.Join(dir, "stdlib")); err == nil {
				return dir
			}
			dir = filepath.Dir(dir)
		}
	}
	// Fall back to working directory
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 5; i++ {
			if _, err := os.Stat(filepath.Join(dir, "stdlib")); err == nil {
				return dir
			}
			dir = filepath.Dir(dir)
		}
	}
	return "."
}

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&File, "file", "f", "", "Ruby file to execute (defaults to stdin)")
}

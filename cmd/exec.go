/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
		compiled, err := compiler.Compile(program)
		if err != nil {
			fmt.Println(err)
			if compiled != "" {
				fmt.Println(compiled)
			}
			return
		}
		if strings.Contains(compiled, "github.com/redneckbeard/thanos/stdlib") {
			if _, err := os.Open("go.mod"); err != nil {
				color.Red("Generated Go source has a dependency on github.com/redneckbeard/thanos/stdlib, but the current directory has no go.mod file. Run `go mod init $mymodule` and `go get github.com/redneckbeard/thanos/stdlib@latest` and try again.")
				return
			}
		}
		goTmp, _ := os.Create("tmp.go")
		defer os.Remove(goTmp.Name())
		goTmp.WriteString(compiled)

		run := exec.Command("go", "run", goTmp.Name())
		var stderr bytes.Buffer
		run.Stderr = &stderr
		stdout, err := run.Output()
		if err != nil {
			color.Red(`Execution failed for compiled Go:
------
%s
------
Error: %s`, compiled, stderr.String())
		} else {
			color.Green(strings.Repeat("-", 20))
			fmt.Print(string(stdout))
		}
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&File, "file", "f", "", "Ruby file to execute (defaults to stdin)")
}

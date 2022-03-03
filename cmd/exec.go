/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"

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
		program, err := parser.ParseFile(Source)
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
		goTmp, _ := os.Create("tmp.go")
		defer os.Remove(goTmp.Name())
		goTmp.WriteString(compiled)

		run := exec.Command("go", "run", goTmp.Name())
		stdout, err := run.Output()
		if err != nil {
			fmt.Printf(`Execution failed for compiled Go:
------
%s
------
Error: %s`, compiled, err.Error())
		} else {
			fmt.Print(string(stdout))
		}
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&File, "file", "f", "", "Ruby file to execute (defaults to stdin)")
}

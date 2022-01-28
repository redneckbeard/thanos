/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/redneckbeard/thanos/compiler"
	"github.com/redneckbeard/thanos/parser"
	"github.com/spf13/cobra"
)

var TestDir, TestFile, TestCase string

func runTest(script, name string) bool {
	fmt.Printf("Running test '%s': ", name)
	if script == "" {
		color.Red("FAIL\n    ")
		color.Red("No Ruby source detected")
		return false
	}
	if diff, compiled, err := compiler.CompareThanosToMRI(script, name); err != nil {
		color.Red("FAIL\n    ")
		color.Red(err.Error())
		return false
	} else if diff != "" {
		color.Red(`FAIL
   
%s
Translation:
------------
%s`, diff, compiled)
		return false
	} else {
		color.Green("PASS")
		return true
	}
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "runs thanos integration tests",
	Long: `Runs the thanos integration suite. The test runner loads all files in the
	test directory and consumes all 'gauntlet("<test name") { <test body> }'
	calls. The test runner loads the test for the test name provided (or all
	tests if no name is given), executes it using system Ruby, transpiles and
	executes it using system Go, and then compares the resulting stdout.`,
	Run: func(cmd *cobra.Command, args []string) {
		tests := map[string]string{}
		testFiles, err := filepath.Glob(filepath.Join(TestDir, "*"))
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, file := range testFiles {
			if TestFile == "" || file == filepath.Join(TestDir, TestFile) {
				// very possible that we're doing something in the test that is _not_
				// valid Ruby like declaring a constant, so ignore errors this time
				// around and report them when we've extracted the body of the gauntlet
				// block
				program, _ := parser.ParseFile(file)
				for _, call := range program.MethodSetStack.Peek().Calls["gauntlet"] {
					tests[strings.Trim(call.Args[0].String(), `"'`)] = call.RawBlock
				}
			}
		}
		if TestCase != "" {
			if script, ok := tests[TestCase]; ok {
				runTest(script, TestCase)
			} else {
				fmt.Println("Could not find test:", TestCase)
			}
		} else {
			var passes, fails int
			for name, script := range tests {
				if runTest(script, name) {
					passes++
				} else {
					fails++
				}
			}
			summary := fmt.Sprintf("\n%d passing tests, %d failures\n", passes, fails)
			if fails > 0 {
				color.Red(summary)
			} else {
				color.Green(summary)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().StringVarP(&TestDir, "dir", "d", "tests", "Directory where gauntlet tests are located")
	testCmd.Flags().StringVarP(&TestFile, "file", "f", "", "Single file relative to test directory from which tests are loaded (default loads all files)")
	testCmd.Flags().StringVarP(&TestCase, "gauntlet", "g", "", "Runs only the gauntlet test with the given name")
}

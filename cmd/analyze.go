package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/redneckbeard/thanos/parser"
	"github.com/spf13/cobra"
)

var analyzeSource string
var analyzeProcess bool

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Show type inference results",
	Long: `Analyze Ruby source and display type inference results.

By default, prints the source annotated with inferred types for all
variables, methods, and constants.

With --process, prints a trace of the analysis pipeline showing what
was visited in which order and how types were determined.`,
	Run: func(cmd *cobra.Command, args []string) {
		if analyzeProcess {
			parser.Tracer = parser.NewTracer()
		}

		if analyzeSource == "" {
			color.Green("Input your Ruby and analyze with Ctrl-D.")
		}
		program, err := parser.ParseFile(analyzeSource)

		if analyzeProcess {
			parser.Tracer.WriteProcess(os.Stdout)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nAnalysis error: %v\n", err)
			}
			return
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}

		// Read source for annotation mode
		var source []byte
		if analyzeSource != "" {
			source, err = os.ReadFile(analyzeSource)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading source: %v\n", err)
				return
			}
		}

		parser.WriteAnnotations(os.Stdout, program, source)
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVarP(&analyzeSource, "source", "s", "", "Ruby source file to analyze")
	analyzeCmd.Flags().BoolVar(&analyzeProcess, "process", false, "Show analysis process trace instead of type annotations")
}

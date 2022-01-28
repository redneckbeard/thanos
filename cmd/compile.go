package cmd

import (
	"os"
	"strings"

	"github.com/alecthomas/chroma/quick"
	"github.com/fatih/color"
	"github.com/redneckbeard/thanos/compiler"
	"github.com/redneckbeard/thanos/parser"
	"github.com/spf13/cobra"
)

var Target, Source string

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Convert Ruby to Go",
	Long:  `Compile source Ruby to Go to the best of thanos's ability. Lacking functionality is described at https://github.com/redneckbeard/thanos#readme and https://github.com/redneckbeard/thanos/issues`,
	Run: func(cmd *cobra.Command, args []string) {
		if Source == "" {
			color.Green("Input your Ruby and compile with Ctrl-D.")
		}
		program, err := parser.ParseFile(Source)
		if err != nil {
			color.Red(err.Error())
			return
		}
		compiled, _ := compiler.Compile(program)
		if Target == "" {
			color.Green(strings.Repeat("-", 20))
			quick.Highlight(os.Stdout, compiled, "go", "terminal256", "monokai")
		} else {
			err = os.WriteFile(Target, []byte(compiled), 0644)
			if err != nil {
				color.Red(err.Error())
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(compileCmd)
	compileCmd.Flags().StringVarP(&Target, "target", "t", "", "Destination for resulting Go (defaults to stdout)")
	compileCmd.Flags().StringVarP(&Source, "source", "s", "", "Destination for resulting Go (defaults to stdin)")
}

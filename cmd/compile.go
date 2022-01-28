package cmd

import (
	"fmt"
	"os"

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
		program, err := parser.ParseFile(Source)
		if err != nil {
			fmt.Println(err)
			return
		}
		compiled, _ := compiler.Compile(program)
		if Target == "" {
			fmt.Println(compiled)
		} else {
			err = os.WriteFile(Target, []byte(compiled), 0644)
			if err != nil {
				fmt.Println(err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(compileCmd)
	compileCmd.Flags().StringVarP(&Target, "target", "t", "", "Destination for resulting Go (defaults to stdout)")
	compileCmd.Flags().StringVarP(&Source, "source", "s", "", "Destination for resulting Go (defaults to stdin)")
}

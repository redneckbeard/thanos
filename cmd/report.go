/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/redneckbeard/thanos/compiler"
	"github.com/redneckbeard/thanos/types"
	"github.com/spf13/cobra"
)

var className string

func report(className string) {
	script := fmt.Sprintf(`methods = %s.instance_methods - Object.instance_methods
methods.sort!.each {|m| puts m}`, className)
	if diff, _, err := compiler.CompareThanosToMRI(script, className); err != nil {
		panic(err)
	} else {
		fmt.Printf("# Methods missing on %s\n\n", className)
		fmt.Printf("The following instance methods have not yet been implemented on %s. This list does not include methods inherited from `Object` or `Kernel` that are missing from those ancestors.\n\n", className)
		for _, method := range strings.Split(diff, "\n") {
			fmt.Printf("* `%s#%s`\n", className, method)
		}
	}
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a report on methods missing from built-in types",
	Long:  `Generates a report on methods missing from built-in types. This currently deliberately excludes TrueClass and FalseClass because of how the thanos type inference framework handles booleans.`,
	Run: func(cmd *cobra.Command, args []string) {
		if className != "" {
			types.ClassRegistry.Initialize()
			if _, err := types.ClassRegistry.Get(className); err != nil {
				fmt.Printf("Class '%s' not found in thanos class registry.\n", className)
			} else {
				report(className)
			}
		} else {
			for i, name := range types.ClassRegistry.Names() {
				if name != "Kernel" && name != "Boolean" {
					report(name)
					if i < len(types.ClassRegistry.Names())-1 {
						fmt.Println("")
					}
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().StringVarP(&className, "class", "c", "", "Ruby class report will be generated for (defaults to all currently implemented core classes")
}

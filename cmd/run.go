/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pl "github.com/zph/polylint/pkg"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [files or folder]",
	Short: "Run linter",
	Long: `Run linter against files or root folder(s)
and usage of using your command. For example:
> polylint --config .polylint run src/ lib/
`,
	Args: cobra.MinimumNArgs(1),
	Run:  RunCmd,
}

func Run(cmd *cobra.Command, args []string) (int, []error) {
	var exitCode int
	exitCode = 0
	var errs []error
	var results []pl.FileReport
	for _, root := range args {
		configRaw, err := os.ReadFile(viper.ConfigFileUsed())

		if err != nil {
			panic(err)
		}
		cfg, err := pl.LoadConfigFile(string(configRaw))

		if err != nil {
			fmt.Printf("error loading config: %v\n", err)
			return exitCode, []error{err}
		}

		err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("error accessing path %q: %v\n", path, err)
				return err
			}

			// Causes 4x slowdown in benchmarks on project with large node_modules folder
			// if we use IsRegular() instead of isDir()
			// Regular mode = non-dir, non-symlink etcs
			// TODO(zph) investigate this issue and find a better solution
			if !info.IsDir() {
				content, err := os.ReadFile(path)

				if err != nil {
					fmt.Printf("error reading file %q: %v\n", path, err)
					return err
				}

				result, err2 := pl.ProcessFile(string(content), path, cfg)
				if err2 != nil {
					return fmt.Errorf("error processing file %q: %v", path, err2)
				}

				results = append(results, result)
			}

			return nil
		})
		errs = append(errs, err)
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"File", "#", "Scope", "Rule Id", "Recommendation", "Link"})

	summary := table.NewWriter()
	summary.SetOutputMirror(os.Stdout)
	summary.AppendHeader(table.Row{"File", "Violations"})
	summary.SortBy([]table.SortBy{
		{Name: "Violations", Mode: table.Dsc},
	})
	for _, result := range results {
		if len(result.Findings) > 0 {
			summary.AppendRow([]interface{}{result.Path, len(result.Findings)})
			for idx, finding := range result.Findings {
				var scope string
				if finding.Rule.Scope == "file" || finding.Rule.Scope == "path" {
					scope = fmt.Sprintf("%s", finding.Rule.Scope)
				} else {
					scope = fmt.Sprintf("%s %3d", finding.Rule.Scope, finding.LineNo)
				}
				t.AppendRow([]interface{}{
					result.Path, idx + 1, scope, finding.RuleId, finding.Rule.Recommendation, finding.Rule.Link,
				})
				exitCode += 1
			}
		}
	}
	summary.Render()
	t.Render()

	if exitCode > 255 {
		exitCode = 255
	}

	nonNilErrors := make([]error, 0)
	for _, e := range errs {
		if e != nil {
			nonNilErrors = append(nonNilErrors, e)
		}
	}
	return exitCode, nonNilErrors
}

func RunCmd(cmd *cobra.Command, args []string) {
	exitCode, errs := Run(cmd, args)
	if len(errs) > 0 {
		fmt.Printf("Errors: %+v", errs)
	}
	os.Exit(exitCode)
}

func init() {
	rootCmd.AddCommand(runCmd)
}

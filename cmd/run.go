/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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

			// Check if the file has the extension
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

				if len(result.Findings) > 0 {
					fmt.Printf("\n%s: violations count %d\n", result.Path, len(result.Findings))
					for idx, finding := range result.Findings {
						// TODO: figure out why the rule embedded is wrong
						fmt.Printf("%d: Line %3d %30s %20s\n", idx+1, finding.LineNo, finding.RuleId, finding.Rule.Description)
					}
					exitCode = 1
				}
			}

			return nil
		})
		errs = append(errs, err)
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

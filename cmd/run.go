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
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.MinimumNArgs(1),
	Run:  RunCmd,
}

func Run(cmd *cobra.Command, args []string) (int, []error) {
	var exitCode int
	exitCode = 0
	var errs []error
	for _, root := range args {
		// Resolve symlinks
		s, err := os.Readlink(viper.ConfigFileUsed())
		if err == nil {
			cfgFile = s
		}
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
						fmt.Printf("%d: %s:%d %s %s\n", idx+1, result.Path, finding.LineNo, finding.RuleId, finding.Rule.Description)
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

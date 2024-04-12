/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pkg "github.com/zph/polylint/pkg"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validation configuration files",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: standardize this to also do symlinks
		content, err := os.ReadFile(viper.ConfigFileUsed())
		if err != nil {
			panic(err)
		}
		cfg, err := pkg.LoadConfigFile(string(content))
		if err != nil {
			panic(err)
		}
		err = pkg.ValidateConfigFile(cfg)
		if err != nil {
			panic(err)
		}
		fmt.Println("validation success")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

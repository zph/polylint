/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version = "v0.0.1"
	commit  = ""
	date    = ""
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "polylint",
	Short: "Polylint: Extensible generalized linter",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(setVersion)
	rootCmd.SetVersionTemplate(fmt.Sprintf("polylint\nVersion: %s\nCommit: %s\nDate: %s\n", version, commit, date))

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.polylint.yaml)")
}

func setVersion() {
	viper.Set("binary_version", version)
	rootCmd.Version = version
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".polylint" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".polylint")
	}

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading config file:", viper.ConfigFileUsed())
	}
}

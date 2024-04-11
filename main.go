package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func Run(args []string) error {
	root := args[0]
	configRaw := args[1]

	cfg, err := LoadConfigFile(configRaw)

	if err != nil {
		fmt.Printf("error loading config: %v\n", err)
		return err
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

			result, err2 := ProcessFile(string(content), path, cfg)
			if err2 != nil {
				return fmt.Errorf("error processing file %q: %v", path, err2)
			}

			if len(result.Findings) > 0 {
				fmt.Printf("%s: violations count %d\n", result.Path, len(result.Findings))
			}
		}

		return nil
	})

	return err
}

func main() {
	err := Run(os.Args[1:])
	if err != nil {
		fmt.Printf("error running the tool: %v\n", err)
		os.Exit(1)
	}
}

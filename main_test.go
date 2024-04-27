package main

import (
	"os"
	"testing"

	pl "github.com/zph/polylint/pkg"
)

const (
	forFileIgnore = `# polylint disable-for-file=no-print,no-print-js
print("A")`

	nextLineIgnore = `# polylint disable-next-line=no-print,no-print-js
print("A")`

	nextLineIgnoreShorthand = `# polylint disable=no-print,no-print-js
print("A")
logging()
`

	nextLineIgnoreDoesntApply = `# polylint disable-next-line=no-print,no-print-js


print("A")`

	fileWithFaultyIgnoreStatement = `# polylint disable-xyz=no-print,no-print-js
print("A") `
)

func TestProcessFile(t *testing.T) {
	simpleConfigFile, err := loadTestingConfigFile(simpleConfigFilePath)
	if err != nil {
		panic(err)
	}
	var tests = []struct {
		name          string
		content       string
		path          string
		findingsCount int
		expectedErr   error
	}{
		{"Basic test without ignores", `print("A")`, "example.py", 5, nil},
		{"Basic test for-file ignore", forFileIgnore, "example.py", 3, nil},
		{"Basic test next-line ignore", nextLineIgnore, "example.py", 3, nil},
		{"Basic test next-line ignore shorthand", nextLineIgnoreShorthand, "example.py", 4, nil},
		{"Basic test next-line ignore doesn't apply", nextLineIgnoreDoesntApply, "example.py", 5, nil},
		{"Basic test with faulty ignore statement", fileWithFaultyIgnoreStatement, "example.py", 5, nil},
		{"Basic test with banned filename", nextLineIgnore, "print.py", 4, nil},
	}
	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := pl.LoadConfigFile(simpleConfigFile)
			if err != nil {
				t.Fatalf("Error loading config file: %v", err)
			}

			result, err := pl.ProcessFile(tt.content, tt.path, cfg)
			if err != tt.expectedErr {
				t.Errorf("Test #%d Error was incorrect, got: %v, want: %v.", idx, err, tt.expectedErr)
			}
			if len(result.Findings) != tt.findingsCount {
				t.Errorf("Test #%d Result was incorrect, findings count: got: %d, want: %d.\n", idx, len(result.Findings), tt.findingsCount)
			}
		})
	}
}

var simpleConfigFilePath = "./examples/simple.yaml"

func loadTestingConfigFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func TestConfigFileParsing(t *testing.T) {
	simpleConfigFile, err := loadTestingConfigFile(simpleConfigFilePath)
	if err != nil {
		panic(err)
	}
	var tests = []struct {
		name              string
		content           string
		expectedRuleCount int
	}{
		{"basic config file with 1 rule", simpleConfigFile, 7},
	}
	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := pl.LoadConfigFile(tt.content)
			if len(result.Rules) != tt.expectedRuleCount {
				t.Errorf("Test #%d Result was incorrect, findings count: got: %d, want: %d.\n", idx, len(result.Rules), tt.expectedRuleCount)
			}
		})
	}
}

package polylint

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
)

type SeverityLevel int
type Scope string
type FnType string

const (
	unknownSeverity SeverityLevel = iota
	lowSeverity
	mediumSeverity
	highSeverity
)

const (
	builtinType FnType = "builtin"
	jsType      FnType = "js"
	wasmType    FnType = "wasm"
)

const (
	unknownScope Scope = "unknown"
	pathScope    Scope = "path"
	fileScope    Scope = "file"
	lineScope    Scope = "line"
)

type Ignore struct {
	Id           string
	Scope        Scope
	SourceLineNo int
	LineNo       int
}

type FileReport struct {
	Path     string
	Ignores  []Ignore
	Rules    []Rule
	Findings []Finding
}

type Finding struct {
	Path      string
	Line      string
	LineIndex int
	LineNo    int
	Rule      Rule
	RuleId    string
}

type RuleFunc func(string, int, string) bool
type Rule struct {
	Fn             RuleFunc
	Id             string
	Description    string
	Recommendation string
	Severity       SeverityLevel
	// Link to the documentation for this Rule
	Link string
	// FilenameRegex is the regexp used to determine if this Rule should run on a given file
	IncludePaths *regexp.Regexp
	ExcludePaths *regexp.Regexp
	Scope        Scope
}

type Fn struct {
	Type  FnType
	Scope Scope
	Name  string
	Args  []any
	Body  string
}

type RawFn struct {
	Type  string
	Scope Scope
	Name  string
	Args  []any
	Body  string

	// sha256: sha256 hash in hex form
	Metadata map[string]any
}

func (f RawFn) GetMetadataHash() (string, error) {
	hash, ok := f.Metadata["sha256"]

	if !ok {
		return "", fmt.Errorf("could not get sha256 hash from metadata for: %s", f.Body)
	}

	h, ok := hash.(string)

	if !ok {
		return "", fmt.Errorf("could not get sha256 hash from metadata for: %s", f.Body)
	}
	return h, nil
}

func (f RawFn) GetWASMFromUrl(url string) ([]byte, error) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Write the body to file
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

func (f RawFn) GetWASMFromPath(path string) ([]byte, error) {
	// Create the file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (f RawFn) WriteWASMToCache(content []byte) (bool, error) {
	// Make dir in ~/.local/cache/polylint/cache/SHA256
	h := sha256.New()

	h.Write(content)

	bs := h.Sum(nil)

	hex := fmt.Sprintf("%x", bs)
	output_folder, err := f.CacheDirWASM()
	if err != nil {
		return false, err
	}
	os.MkdirAll(output_folder, 0755)
	os.WriteFile(path.Join(output_folder, hex), content, 0755)
	return true, nil
}

func (f RawFn) CacheDirWASM() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return path.Join(home, ".local", "cache", "polylint", "cache"), nil
}

func (f RawFn) CheckWASMHash(content []byte, hash string) bool {
	h := sha256.New()

	h.Write(content)

	bs := h.Sum(nil)

	actual := fmt.Sprintf("%x", bs)
	comparison := actual == hash
	if !comparison {
		logz.Infof("Comparison failed for desired sha256 %s and actual: %s\n", hash, actual)
	}
	return comparison
}

func (f RawFn) GetWASMFromCache(hash string) ([]byte, error) {
	dir, err := f.CacheDirWASM()
	if err != nil {
		return nil, err
	}
	// TODO: move to debug level logging
	logz.Debugf("Success fetching file from cache %s\n", hash)
	return os.ReadFile(path.Join(dir, hash))
}

type RawRule struct {
	Id             string
	Description    string
	Recommendation string
	Severity       string
	Link           string
	IncludePaths   *regexp.Regexp `yaml:"include_paths"`
	ExcludePaths   *regexp.Regexp `yaml:"exclude_paths"`
	Fn             RawFn
}

type RawConfig struct {
	Version  string
	Includes []IncludeRaw
	Rules    []RawRule
}

type IncludeRaw struct {
	Path string `yaml:"path"`
	// Hash of the file contents prefixed with the algorithm
	Hash string `yaml:"hash"`
}

type ConfigFile struct {
	Rules   []Rule
	Version string
}

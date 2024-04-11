package main

import (
	"regexp"
)

type SeverityLevel int
type Scope string

const (
	unknownSeverity SeverityLevel = iota
	lowSeverity
	mediumSeverity
	highSeverity
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
	Rule      *Rule
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
	Type  string
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
	// SHA256 of the file contents
	SHA string `yaml:"sha"`
}

type ConfigFile struct {
	Rules   []Rule
	Version string
}

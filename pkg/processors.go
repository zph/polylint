package polylint

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dop251/goja"
	extism "github.com/extism/go-sdk"
	"github.com/spf13/viper"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
)

// TODO: use more efficient data structure... like map[int]Ignore where int = line number
func extractIgnoresFromLine(line string, lineNo int, f *FileReport) error {
	if strings.Contains(line, "polylint disable") {
		chunks := strings.SplitN(line, "=", 2)
		directive := strings.TrimSpace(strings.SplitN(chunks[0], "polylint", 2)[1])
		ignoresStr := strings.TrimSpace(chunks[1])
		if directive == "disable-for-file" {
			ignores := strings.Split(ignoresStr, ",")
			for _, ignore := range ignores {
				f.Ignores = append(f.Ignores, Ignore{Scope: fileScope, SourceLineNo: lineNo, LineNo: 0, Id: ignore})
			}
		} else if directive == "disable-next-line" || directive == "disable-line" || directive == "disable" {
			ignores := strings.Split(ignoresStr, ",")
			for _, ignore := range ignores {
				nextLineNo := lineNo + 1
				f.Ignores = append(f.Ignores, Ignore{Scope: lineScope, SourceLineNo: lineNo, LineNo: nextLineNo, Id: ignore})
			}
		} else if directive == "disable-for-path" {
			ignores := strings.Split(ignoresStr, ",")
			for _, ignore := range ignores {
				f.Ignores = append(f.Ignores, Ignore{Scope: pathScope, SourceLineNo: lineNo, LineNo: 0, Id: ignore})
			}
		} else {
			logz.Warnf("WARNING: directive for polylint not recognized on line %d %s %s\n", lineNo, directive, ignoresStr)
			return nil
		}
	}
	return nil
}

func getIgnoresForLine(f *FileReport, lineNo int) map[string]Ignore {
	var ignores = make(map[string]Ignore)
	for _, ignore := range f.Ignores {
		if ignore.Scope == lineScope && ignore.LineNo == lineNo {
			ignores[ignore.Id] = ignore
		} else if ignore.Scope == fileScope {
			ignores[ignore.Id] = ignore
		}
	}
	return ignores
}

func processLine(line string, idx int, f *FileReport) error {
	lineNo := idx + 1
	if err := extractIgnoresFromLine(line, lineNo, f); err != nil {
		return err
	}
	ignores := getIgnoresForLine(f, lineNo)

	// Ignore declaration lines of lint rules which requires that we not support
	// end of line polylint ignore declarations
	if strings.Contains(line, "polylint") {
		return nil
	}

	for _, rule := range f.Rules {
		if rule.Scope != lineScope {
			continue
		}
		// Start with strictest, which is denies
		if rule.ExcludePaths != nil && rule.ExcludePaths.MatchString(f.Path) {
			continue
		}
		if rule.IncludePaths != nil && rule.IncludePaths.MatchString(f.Path) {
			if _, ok := ignores[rule.Id]; !ok {
				if rule.Fn(f.Path, idx, line) {
					finding := Finding{Path: f.Path, LineNo: lineNo, LineIndex: idx, Line: line, Rule: rule, RuleId: rule.Id}
					f.Findings = append(f.Findings, finding)
				}
			}
		}
	}
	return nil
}

func LoadConfigFile(content string) (ConfigFile, error) {
	var rawConfig RawConfig
	var config ConfigFile
	err := yaml.Unmarshal([]byte(content), &rawConfig)
	if err != nil {
		logz.Errorf("Error unmarshalling YAML: %v", err)
		return ConfigFile{}, err
	}

	if !strings.HasPrefix(rawConfig.Version, "v") {
		logz.Errorf("Error: config file version must start with a 'v' but was %s\n", rawConfig.Version)
		panic("Invalid version due to semver incompatibility")
	}

	if !semver.IsValid(rawConfig.Version) {
		logz.Errorf("Error: Config version %s is newer than binary version %s\n", rawConfig.Version, viper.GetString("binary_version"))
		logz.Errorln(semver.IsValid(rawConfig.Version))
		panic("Invalid version due to semver incompatibility")
	}

	// If version file is too new for binary version
	if semver.Compare(rawConfig.Version, viper.GetString("binary_version")) == 1 {
		_ = 1
		// TODO: determine how to handle version for version check when not set in tests
		// Ignore for now until we can control the output
		//logz.Warnf("Warning: config file version %s is newer than binary version %s\n", rawConfig.Version, viper.GetString("binary_version"))
	}

	config.Version = rawConfig.Version
	for _, rule := range rawConfig.Rules {
		config.Rules = append(config.Rules, Rule{
			Id:             rule.Id,
			Description:    rule.Description,
			Recommendation: rule.Recommendation,
			Severity:       severityLevelFromString(rule.Severity),
			Link:           rule.Link,
			IncludePaths:   rule.IncludePaths,
			ExcludePaths:   rule.ExcludePaths,
			Fn:             BuildFn(rule.Fn),
			Scope:          rule.Fn.Scope,
		})
	}

	for _, include := range rawConfig.Includes {
		u, err := url.Parse(include.Path)
		if err != nil {
			panic(fmt.Sprintf("Error parsing include path %s", include.Path))
		}

		var content string
		switch u.Scheme {
		case "":
			content = getFileContentFromURI(u)
		case "file":
			content = getFileContentFromURI(u)
		case "http", "https":
			content = getURLContent(u.String())
		}

		// TODO: add tests
		if include.Hash != "" {
			if !CheckContentHash(include.Hash, content) {
				panic(fmt.Sprintf("WARNING: content does not match expected hash.\n Actual %s!= Expected %s\n", include.Hash, "sha256:"+SHA256(content)))
			}
		}

		cfg, err := LoadConfigFile(content)
		if err != nil {
			panic(err)
		}
		config.Rules = append(config.Rules, cfg.Rules...)
	}

	return config, nil
}

func CheckContentHash(expectedHash, content string) bool {
	chunks := strings.Split(expectedHash, ":")
	var algo string
	var hash string

	algo = "sha256"
	if len(chunks) == 2 {
		algo = strings.ToLower(chunks[0])
		hash = chunks[1]
	} else {
		hash = expectedHash
	}

	switch algo {
	case "sha256":
		return SHA256(content) == hash
	default:
		return SHA256(content) == hash
	}
}

func ValidateConfigFile(config ConfigFile) error {
	// Ensure uniqueness of rule ids
	ids := make(map[string]bool)
	for _, rule := range config.Rules {
		if _, ok := ids[rule.Id]; !ok {
			ids[rule.Id] = true
		} else {
			panic(fmt.Sprintf("Received duplicate id %s", rule.Id))
		}
	}
	return nil
}

func getFileContentFromURI(u *url.URL) string {
	absPath, err := filepath.Abs(u.Path)
	if err != nil {
		panic(fmt.Sprintf("Error getting absolute path %s: %v", u.Path, err))
	}
	fileContent, err := os.ReadFile(absPath)
	if err != nil {
		panic(fmt.Sprintf("Error reading file %s: %v", u.Path, err))
	}
	return string(fileContent)
}

func SHA256(content string) string {
	h := sha256.New()

	h.Write([]byte(content))

	bs := h.Sum(nil)

	return fmt.Sprintf("%x", bs)
}

func getURLContent(link string) string {
	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	content, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

func severityLevelFromString(s string) SeverityLevel {
	switch s {
	case "low":
		return lowSeverity
	case "medium":
		return mediumSeverity
	case "high":
		return highSeverity
	default:
		panic("unknown severity level")
	}
}

func buildLineFnBuiltin(f RawFn) RuleFunc {
	switch f.Name {
	case "contains":
		matchOn := f.Args[0].(string)
		return func(_path string, _idx int, line string) bool {
			return strings.Contains(line, matchOn)
		}
	case "regexp":
		matchOnRaw := f.Args[0].(string)
		matchOn := regexp.MustCompile(matchOnRaw)
		return func(_path string, _idx int, line string) bool {
			return matchOn.MatchString(line)
		}
	default:
		panic(fmt.Sprintf("unknown builtin %s", f.Name))
	}
}

func BuildFn(f RawFn) RuleFunc {
	switch f.Scope {
	case lineScope:
		return BuildLineFn(f)
	case fileScope:
		return BuildFileScopeFn(f)
	case pathScope:
		return BuildPathScopeFn(f)
	default:
		panic(fmt.Sprintf("unknown scope %s", f.Scope))
	}
}

func BuildLineFn(f RawFn) RuleFunc {
	switch f.Type {
	case "builtin":
		return buildLineFnBuiltin(f)
	case "js":
		return buildJsFn(f)
	case "wasm":
		return buildWasmFn(f)
	default:
		panic(fmt.Sprintf("unknown type %s", f.Type))
	}
}

func BuildFileScopeFn(f RawFn) RuleFunc {
	switch f.Type {
	case "builtin":
		return BuildFileFnBuiltin(f)
	case "js":
		return BuildFileFnJs(f)
	case "wasm":
		return buildWasmFn(f)
	default:
		panic(fmt.Sprintf("unknown type %s", f.Type))
	}
}

func BuildFileFnBuiltin(f RawFn) RuleFunc {
	switch f.Name {
	case "contains":
		matchOn := f.Args[0].(string)
		return func(_path string, _idx int, file string) bool {
			return strings.Contains(file, matchOn)
		}
	case "regexp":
		matchOnRaw := f.Args[0].(string)
		matchOn := regexp.MustCompile(matchOnRaw)
		return func(_path string, _idx int, file string) bool {
			return matchOn.MatchString(file)
		}
	default:
		panic(fmt.Sprintf("unknown builtin %s", f.Name))
	}
}

func BuildFileFnJs(f RawFn) RuleFunc {
	vm := goja.New()
	_, err := vm.RunString(f.Body)
	if err != nil {
		panic(err)
	}
	var fn func(path string, idx int, file string) bool
	err = vm.ExportTo(vm.Get(f.Name), &fn)
	if err != nil {
		panic(err)
	}

	return fn
}

func BuildPathScopeFn(f RawFn) RuleFunc {
	switch f.Type {
	case "builtin":
		return BuildPathFnBuiltin(f)
	case "js":
		return BuildPathFnJs(f)
	case "wasm":
		return buildWasmFn(f)
	default:
		panic(fmt.Sprintf("unknown type %s", f.Type))
	}
}

func BuildPathFnBuiltin(f RawFn) RuleFunc {
	switch f.Name {
	case "contains":
		matchOn := f.Args[0].(string)
		return func(path string, _idx int, _file string) bool {
			return strings.Contains(path, matchOn)
		}
	case "regexp":
		matchOnRaw := f.Args[0].(string)
		matchOn := regexp.MustCompile(matchOnRaw)
		return func(path string, _idx int, _file string) bool {
			return matchOn.MatchString(path)
		}
	default:
		panic(fmt.Sprintf("unknown builtin %s", f.Name))
	}

}

func BuildPathFnJs(f RawFn) RuleFunc {
	vm := goja.New()
	_, err := vm.RunString(f.Body)
	if err != nil {
		panic(err)
	}
	var fn func(path string, idx int, file string) bool
	err = vm.ExportTo(vm.Get(f.Name), &fn)
	if err != nil {
		panic(err)
	}

	return fn
}

func buildJsFn(f RawFn) RuleFunc {
	vm := goja.New()
	_, err := vm.RunString(f.Body)
	if err != nil {
		panic(err)
	}
	var fn func(path string, idx int, line string) bool
	err = vm.ExportTo(vm.Get(f.Name), &fn)
	if err != nil {
		panic(err)
	}

	return fn
}

func buildWasmFn(f RawFn) RuleFunc {
	hash, err := f.GetMetadataHash()
	if err != nil {
		logz.Warnf("Warning: cannot find metadata hash for %s\n", f.Body)
	}
	var content []byte

	// TODO: handle null case of hash
	if hash != "" {
		content, err = f.GetWASMFromCache(hash)
	}
	if err != nil {
		if strings.HasPrefix(f.Body, "http") {
			content, err = f.GetWASMFromUrl(f.Body)
		} else {
			content, err = f.GetWASMFromPath(f.Body)
		}
	}
	if err != nil {
		panic(err)
	}

	ok := f.CheckWASMHash(content, hash)
	if !ok && hash != "" {
		logz.Errorf("hash mismatch for %s", f.Body)
	}
	f.WriteWASMToCache(content)

	var location []extism.Wasm
	location = append(location, extism.WasmData{
		Data: content,
		Hash: hash,
	})

	manifest := extism.Manifest{
		Wasm: location,
	}

	ctx := context.Background()
	config := extism.PluginConfig{
		EnableWasi: true,
	}

	plugin, err := extism.NewPlugin(ctx, manifest, config, []extism.HostFunction{})
	if err != nil {
		logz.Errorf("Failed to initialize plugin: %v\n", err)
		os.Exit(1)
	}

	return func(path string, idx int, line string) bool {
		args := RuleFuncArgs{path, idx, line}
		b, err := json.Marshal(&args)
		if err != nil {
			panic(err)
		}

		exit, bytes, err := plugin.CallWithContext(ctx, f.Name, b)
		if err != nil {
			logz.Errorln(err)
			os.Exit(int(exit))
		}
		var result RuleFuncResult
		json.Unmarshal(bytes, &result)

		return result.Value
	}
}

type RuleFuncArgs [3]interface{}

type RuleFuncResult struct {
	Value bool
}

func ProcessFile(content string, path string, cfg ConfigFile) (FileReport, error) {
	f := FileReport{
		Path:     path,
		Rules:    cfg.Rules,
		Findings: []Finding{},
	}

	lines := strings.Split(content, "\n")
	for idx, line := range lines {
		err := processLine(line, idx, &f)
		if err != nil {
			logz.Errorf("ERROR: %s\n", err)
			return FileReport{}, err
		}
	}

	ignores := getIgnoresForLine(&f, 0)
	for _, c := range f.Rules {
		if c.Scope == fileScope || c.Scope == pathScope {
			lineIdx := -1
			lineNo := 0
			if c.ExcludePaths != nil && c.ExcludePaths.MatchString(f.Path) {
				continue
			}
			// TODO: currently does not support checking for file level ignores or path ignores
			if c.IncludePaths != nil && c.IncludePaths.MatchString(f.Path) {
				if _, ok := ignores[c.Id]; !ok {
					if c.Fn(f.Path, lineIdx, content) {
						f.Findings = append(f.Findings, Finding{
							Path:      f.Path,
							Line:      content,
							LineIndex: lineIdx,
							LineNo:    lineNo,
							Rule:      c,
							RuleId:    c.Id,
						})
					}
				}
			}
		}
	}

	return f, nil
}

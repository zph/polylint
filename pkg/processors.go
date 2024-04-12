package polylint

import (
	"crypto/sha256"
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
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
)

// TODO: use more efficient data structure... like map[int]Ignore where int = line number
func extractIgnoresFromLine(line string, lineNo int, f *FileReport) error {
	if strings.Contains(line, "polylint disable") {
		chunks := strings.SplitN(line, "=", 2)
		// chunks[0] = `# polylint disable.*` up to the equals sign
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
			fmt.Printf("WARNING: directive for polylint not recognized on line %d %s %s\n", lineNo, directive, ignoresStr)
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
		fmt.Printf("Error unmarshalling YAML: %v", err)
		return ConfigFile{}, err
	}

	if !strings.HasPrefix(rawConfig.Version, "v") {
		fmt.Printf("Error: config file version must start with a 'v' but was %s\n", rawConfig.Version)
		panic("Invalid version due to semver incompatibility")
	}

	if !semver.IsValid(rawConfig.Version) {
		fmt.Printf("Error: Config version %s is newer than binary version %s\n", rawConfig.Version, PolylintVersion)
		fmt.Println(semver.IsValid(rawConfig.Version))
		panic("Invalid version due to semver incompatibility")
	}

	// If version file is too new for binary version
	if semver.Compare(rawConfig.Version, PolylintVersion) == 1 {
		fmt.Printf("Warning: config file version %s is newer than binary version %s\n", rawConfig.Version, PolylintVersion)
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
			fmt.Printf("ERROR: %s\n", err)
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

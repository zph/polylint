# PolyLint - Fast and Extendable Generic Linter

# Features

- [ ] Standard linters built using golang functions
  - [ ] contains
  - [ ] regexp match
- [ ] Ignore mechanisms
  - [x] Ignore full file
    - [x] polylint disable-for-file=$RULE_ID
  - [x] Ignore next line
    - [x] polylint disable-next-line=$RULE_ID
    - [x] polylint disable=$RULE_ID
  - [ ] Ignore from lint runner via cli for file-level ignores
- [ ] Types of rules
  - [x] line
  - [ ] file content
  - [ ] file path
- [ ] Builtin linters configured for use in config file
- [ ] Plugin linters configured for use in config file
  - [ ] otto / goja / v8go
  - [ ] supports line / file / path types
- [ ] Use cobra for CLI
- [ ] Replace argv[0] option to accept:
  - a directory
  - many files passed as argv
  - a testing config file used as virtual filesystem
- [ ] Ensure that we confirm uniqueness of rule ids at the  beginning of run during a pre-flight check
- [ ] Build in a way to source rules from remote locations, via path + sha?
  - [ ] ie re-usable plugin infrastructure
  - make it an includes that is cached and pulled at runtime w/ SHAs
- [ ] Rename rules to... rules or validations?
- [ ] Add validation that the version of config file is supported

# Configuration

## Configuration Language

```yaml
---
version: 1.0
rules:
- id: no-print
  description: "Don't use print()"
  recommendation: "Use logging instead."
  severity: low
  link: https://docs.openstack.org/bandit/latest/plugins/b302_no_print.html
  include_paths: '\.py$'
  exclude_paths: null
  fn:
    type: builtin
    scope: line
    name: contains
    args: ['print(']
- id: no-print-js
  description: "Don't use print()"
  recommendation: "Use logging instead."
  severity: low
  link: https://docs.openstack.org/bandit/latest/plugins/b302_no_print.html
  include_paths: '\.py$'
  exclude_paths: null
  fn:
    type: js
    scope: line
    name: printRule
    body: |
      function printRule(path, idx, line) {
        return line.contains('print')
      }
- id: no-print-js-file-level
  description: "Don't use print()"
  recommendation: "Use logging instead."
  severity: low
  link: https://docs.openstack.org/bandit/latest/plugins/b302_no_print.html
  include_paths: '\.py$'
  exclude_paths: null
  fn:
    type: js
    scope: file
    name: printRule
    body: |
      function printRule(path, idx, file) {
        return file.contains('print')
      }
```


# Benchmarks

## 2024-04-08

Using test repo https://github.com/zph/runbook
commit f290434f61a2d2b975cdcdcad060c4e01d2cdfc3 (HEAD -> main, tag: 0.3.0, origin/main, origin/HEAD)

```
❯ hyperfine -- "./bin/polylint ~/src/runbook"
Benchmark 1: ./bin/polylint ~/src/runbook
  Time (mean ± σ):     281.4 ms ±   1.3 ms    [User: 155.5 ms, System: 133.5 ms]
  Range (min … max):   279.1 ms … 282.7 ms    10 runs
```

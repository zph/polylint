# PolyLint - Fast and Extendable Generic Linter

# Features

- Simple and fast golang based builtin linting functions
- Extensible embedded javascript based linters
- Linting configurations can be `included` and referenced from external file or via http(s)
- Each rule contains a severity, path match, path exclusions

# Configuration

## Configuration Language

See [config](examples/simple.yaml)

# Benchmarks

## 2024-04-11

Using test repo https://github.com/zph/runbook
commit f290434f61a2d2b975cdcdcad060c4e01d2cdfc3 (HEAD -> main, tag: 0.3.0, origin/main, origin/HEAD)

```
❯ hyperfine --ignore-failure -- "./bin/polylint --config examples/simple.yaml run ~/src/runbook"
Benchmark 1: ./bin/polylint --config examples/simple.yaml run ~/src/runbook
  Time (mean ± σ):     379.2 ms ±   3.7 ms    [User: 251.0 ms, System: 142.4 ms]
  Range (min … max):   374.4 ms … 384.6 ms    10 runs
```

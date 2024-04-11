# Architecture

## Plugins

### Custom Rules

Custom rules can be added to Polylint by implementing the `Rule` interface and registering it with the plugin manager through a yaml configuration file.

The rule engine supports a basic form of javascript through an embedded js engine in golang.

Each type of rule (line, file, filename) has a different signature:

1. line - `check(path: string, idx: int, line: string) boolean, error`
2. file - `check(content: string) boolean, error`
3. filename - `check(path: string) boolean, error

## Tags for Ignoring Rules

Ignore rules can be declared as:

1. Linewise ignores of the next line (e.g. `polylint disable-next-line=$RULE_ID,$RULE_ID2`)
2. File level ignores (e.g. `polylint disable-for-file=$RULE_ID,$RULE_ID2`)

Polylint does not support suffix based rules for same line because it adds a modest amount of parsing
complexity. If we discover unique needs for inline ignores, it can be added in the future.

Polylint allows for ~any naming of the RULE_IDs and recommends not using the SC1000 style of shellcheck
and instead preferring slug-style-ids (e.g. `polylint disable=no-print`). This is to avoid the lookup
required for humans to understand the rule itself.

Prior Art:
1. How shellcheck does it: https://www.shellcheck.net/wiki/Ignore
2. Prettier linting

# Configuration

```yaml
---
version: 0.0.1
```

## Includes


# Output

Polylint should be able to output the following:
1. json format
2. human format

# Performance is a feature

We build this library first focused on features and second focused on performance.

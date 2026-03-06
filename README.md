# goarch

Architecture-enforcing build proxy for Go. Validates structural rules before allowing compilation.

goarch replaces `go build` with `go tool goarch build` — a command that runs architectural validation first, then compiles. Direct `go build` is blocked at compile time for projects that import the gate package.

## How It Works

```
go tool goarch build ./cmd/api

goarch: validating architecture...
config.go:18:2: [secretguard] sensitive field "OpenRouterAPIKey" should use type Secret
handler/completions.go:114: [execguard] os.Getenv is banned: Use config package

goarch: BUILD BLOCKED — fix architecture violations above.
goarch: Run 'go tool goarch explain <rule>' for details on any rule.
```

When all rules pass:

```
go tool goarch build -o bin/api ./cmd/api

goarch: validating architecture...
goarch: architecture OK
goarch: go build -tags goarch_*** -o bin/api ./cmd/api
```

Direct `go build` produces a compile error:

```
go build ./cmd/api

# github.com/ksanderer/goarch/gate
[goarch] go build is blocked. Use 'go tool goarch build' instead:1: ...
```

## Setup

Two steps in any Go project:

### 1. Add the gate import

```go
// cmd/api/main.go
import _ "github.com/ksanderer/goarch/gate"
```

This blocks direct `go build` at compile time. Only `go tool goarch build` can compile the project.

### 2. Register the tool

```bash
go get github.com/ksanderer/goarch
go mod edit -tool github.com/ksanderer/goarch/cmd/goarch
go mod tidy
```

No separate install needed. `go tool goarch` compiles and caches the binary automatically from `go.mod`.

### 3. Create `.goarch.yml`

```bash
cp .goarch.example.yml .goarch.yml
```

Edit rules to match your project structure. See [Configuration](#configuration) below.

## Commands

```bash
go tool goarch build [flags] [packages]   # Validate → go build
go tool goarch run [flags] [packages]     # Validate → go run
go tool goarch test [flags] [packages]    # Validate → go test
go tool goarch check ./...                # Validate only
go tool goarch explain <rule>             # Show rule documentation
go tool goarch rules                      # List all rules
```

## Analyzers

### layerguard — Layer Isolation

Enforces dependency allowlists/denylists per package. Prevents spaghetti imports.

```yaml
layerguard:
  layers:
    "internal/domain":
      allow: ["internal/logger"]
      deny_all_others: true          # only allow listed + stdlib
    "internal/provider":
      deny: ["internal/handler/*"]   # block specific imports
```

### execguard — Banned Import Guard

Bans specific packages or methods outside allowed locations.

```yaml
execguard:
  banned:
    - pkg: "os/exec"
      except: ["internal/subprocess"]
      reason: "Shell execution only in subprocess"
    - pkg: "os"
      methods: ["Getenv", "Setenv"]
      except: ["internal/config"]
      reason: "Use config package for env vars"
    - pkg: "github.com/sirupsen/logrus"
      reason: "Use slog"
```

### secretguard — Sensitive Field Guard

Ensures struct fields with sensitive names use a wrapper type instead of plain strings. The Secret type should redact `String()`/`GoString()` but **not** `MarshalJSON()` — JSON redaction is handled at the struct level via `json:"-"` or custom marshalers.

Word-boundary aware matching: `"apikey"` matches `OpenRouterAPIKey` but `"token"` does NOT match `PromptTokens`.

```yaml
secretguard:
  type: "Secret"
  field_patterns: ["apikey", "secret", "password", "accesstoken"]
  except_packages: ["internal/handler"]  # response DTOs are OK
```

### fanout — Fan-Out Complexity

Limits non-stdlib imports per file.

```yaml
fanout:
  max_imports: 15
```

### methodcount — Method Count Limit

Limits exported methods per type.

```yaml
methodcount:
  max_public_methods: 20
```

### apileak — API Type Leak Guard

Prevents internal types from appearing in public API function signatures.

```yaml
apileak:
  public_packages: ["internal/api"]
  banned_types_in_public:
    - "internal/executor.*"
    - "internal/statebackend.*"
```

## Configuration

All rules are configured via `.goarch.yml` in the project root. goarch searches up the directory tree for this file. Rules not present in the config are disabled.

See [`.goarch.example.yml`](.goarch.example.yml) for a full example.

## Built-in Documentation

Every violation includes a rule ID. Use `explain` to get detailed docs with fix examples:

```bash
go tool goarch explain secretguard

RULE: secretguard — Sensitive Field Guard

WHAT IT CHECKS:
  Struct fields with names matching sensitive patterns...

FIX:
  1. Create a Secret type in your project...
  2. Change field types...
  3. Use .Value() where the real value is needed...
```

## Architecture

```
goarch/
  gate/                  # Import to block direct go build
    gate_block.go        # Compile error without build tag
    gate_ok.go           # Empty file with build tag (bypass)
  config/                # .goarch.yml loader
  analyzers/             # One package per rule
    layerguard/
    execguard/
    secretguard/
    fanout/
    methodcount/
    apileak/
  docs/                  # Microdocs for explain command
  cmd/goarch/            # CLI: build proxy + multichecker
```

## Design Principles

- **Convention over configuration**: sensible defaults, override via YAML.
- **Config-driven**: adding a rule = adding YAML, not writing code.
- **Fail loud**: compile-time block, not warnings. Build doesn't pass until rules are satisfied.
- **Self-documenting**: every violation has a rule ID, every rule has `explain` docs with fix examples.
- **Zero install**: `go tool` handles compilation and caching from `go.mod`.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for the full plan to reach parity with the Java ArchUnit reference spec.

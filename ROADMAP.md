# goarch Roadmap

## Reference Specification

This project aims to match the architectural enforcement capabilities described in:

**[archunit-rust-go-audit.md](docs/archunit-rust-go-audit.md)** — a per-rule analysis of 84 ArchUnit rules + 6 build tools from a Java project (schedulr), with detailed comparison of how each rule maps to Rust and Go.

That document is the source of truth for what rules exist, why they exist, and how they should be enforced. Every phase below references specific rule IDs from that spec.

---

## Current State

goarch currently has **6 analyzers** covering ~44 of 84 rules (including rules Go covers for free at the language level).

| Analyzer | Spec Rules Covered |
|----------|-------------------|
| `layerguard` | S1-S8, S-parser, S-core, L1-L7 (layer isolation) |
| `execguard` | S12 (process isolation), S23/L12 (no direct env), L10 (sleep ban) |
| `secretguard` | S17 (sensitive field annotations) |
| `fanout` | ClassFanOutComplexity |
| `methodcount` | MethodCount |
| `apileak` | S18 (no internal types in API) |
| Go language | S24/L13 (no cycles), S21 (no field injection), S20 (no annotation magic), S14 (structs=records), L9 (no inheritance), S13 (no codegen mappers) |

---

## Phase 1 — Config-only expansion

**Effort**: 30 min — no new code, only `.goarch.yml` changes.
**Closes**: ~20 rules.

### 1.1 Banned libraries via execguard

Covers spec section 7 (28 banned library rules).

```yaml
execguard:
  banned:
    # Logging — standardize on slog/zerolog
    - pkg: "github.com/sirupsen/logrus"
      reason: "Use structured logging (slog or zerolog)"
    - pkg: "go.uber.org/zap"
      reason: "Use structured logging (slog or zerolog)"
    - pkg: "log4go"
      reason: "Use structured logging"

    # JSON — standardize on encoding/json
    - pkg: "github.com/json-iterator/go"
      reason: "Use encoding/json"
    - pkg: "github.com/buger/jsonparser"
      reason: "Use encoding/json"

    # HTTP clients — standardize on net/http
    - pkg: "github.com/go-resty/resty"
      reason: "Use net/http"
    - pkg: "github.com/parnurzeal/gorequest"
      reason: "Use net/http"

    # Routers — standardize on chi
    - pkg: "github.com/gin-gonic/gin"
      reason: "Use chi"
    - pkg: "github.com/labstack/echo"
      reason: "Use chi"
    - pkg: "github.com/gofiber/fiber"
      reason: "Use chi"

    # Testing — standardize on stdlib testing
    - pkg: "github.com/stretchr/testify"
      reason: "Use stdlib testing"
      # Optional: many Go projects use testify, decide per-project
```

**Spec rules**: Banned JSON (gson equiv), Banned logging (JUL equiv), Banned HTTP clients, Banned routers (Quarkus equiv), Banned test frameworks.

### 1.2 fmt.Print ban

Covers spec rule L11 (no_system_out / no_system_err).

```yaml
    - pkg: "fmt"
      methods: ["Println", "Printf", "Print"]
      except: ["cmd/*", "bench", "internal/streaming"]
      reason: "Use structured logging"
```

### 1.3 time.Sleep ban

Covers spec rule L10 (no_thread_sleep_outside_retry).

```yaml
    - pkg: "time"
      methods: ["Sleep"]
      except: ["internal/retry"]
      reason: "Use retry package for backoff"
```

### 1.4 os/exec ban

Covers spec rule S12 (process_builder_only_in_subprocess).

```yaml
    - pkg: "os/exec"
      except: ["internal/subprocess"]
      reason: "Shell execution only in subprocess package"
```

### 1.5 Activate apileak

Covers spec rule S18 (api_methods_must_not_return_limitr_internals).

```yaml
apileak:
  public_packages: ["internal/api"]
  banned_types_in_public:
    - "internal/pipeline.*"
    - "internal/provider.*"
```

---

## Phase 2 — Simple analyzers

**Effort**: ~1 day total (50-80 lines each).
**Closes**: ~8 rules.

### 2.1 `funlen` — Function length limit

**Spec rule**: Checkstyle MethodLength (max 75 lines).

```yaml
funlen:
  max_lines: 75
```

Implementation: walk `*ast.FuncDecl`, count lines between opening `{` and closing `}`.

### 2.2 `argcount` — Argument count limit

**Spec rule**: Checkstyle ParameterNumber (max 7).

```yaml
argcount:
  max_args: 7
```

Implementation: check `fn.Type.Params.List` length.

### 2.3 `complexity` — Cyclomatic complexity

**Spec rule**: Checkstyle CyclomaticComplexity (max 15).

```yaml
complexity:
  max_complexity: 15
```

Implementation: walk function body, count `if`, `for`, `range`, `case`, `&&`, `||`, `select case`. Start at 1.

### 2.4 `depban` — go.mod dependency ban

**Spec rule**: Maven Enforcer banned artifacts.

Unlike `execguard` (checks imports), `depban` checks `go.mod` directly. Catches transitive dependencies that aren't directly imported.

```yaml
depban:
  deny:
    - module: "github.com/sirupsen/logrus"
    - module: "github.com/gin-gonic/gin"
  max_dependencies: 50  # optional: cap total dep count
```

Implementation: parse `go.mod`, check `require` blocks.

---

## Phase 3 — Medium analyzers

**Effort**: ~2 days total (100-150 lines each).
**Closes**: ~6 rules.

### 3.1 `tagguard` — Struct tag validation

**Spec rules**: S15 (rest_dtos_must_implement_snake_case_dto), S16/L8 (api_records_implement_camel_case_json).

```yaml
tagguard:
  packages:
    "internal/domain":
      json_naming: "snake_case"     # all json tags must be snake_case
      require_json_tags: true       # exported fields must have json tag
    "internal/api":
      json_naming: "camelCase"
```

Implementation: walk struct fields, parse `json:"..."` tags, validate naming convention.

### 3.2 `errguard` — Error type placement

**Spec rule**: S19/L17 (exceptions_in_exception_package).

```yaml
errguard:
  allowed_packages: ["internal/domain"]
```

Implementation: find types implementing `error` interface, check package path.

### 3.3 `authguard` — Endpoint auth coverage

**Spec rule**: S10 (rest_methods_must_have_security_annotation).

This is the hardest to map from Java annotations to Go. In Go, auth is middleware-based, not annotation-based.

Approach: analyze router setup code to verify all non-exempt routes go through auth middleware. This requires understanding chi/mux router patterns.

```yaml
authguard:
  router_package: "cmd/api"
  auth_middleware: "middleware.Auth"
  exempt_patterns: ["/health", "/auth/*", "/webhooks/*"]
```

Implementation: AST analysis of router.Use() and router.Route() chains. Complex but doable.

---

## Phase 4 — Complex analyzers

**Effort**: ~3 days total.
**Closes**: ~6 rules.

### 4.1 `dupl` — Copy-paste detection

**Spec rule**: PMD CPD (50 token minimum) + jscpd.

```yaml
dupl:
  min_tokens: 50
  ignore: ["*_test.go"]
```

Two approaches:
- **Native**: tokenize Go source, suffix array for duplicate detection. ~300 lines.
- **External**: wrap `jscpd` or Go-native `dupl` tool as a subprocess. Simpler.

Recommendation: external integration (Phase 5) is more pragmatic.

### 4.2 `nilguard` — Nil safety checks

**Spec rules**: S25-S26 (no_optional_fields, no_nullable_annotations).

Go's biggest structural gap vs Java+ArchUnit and Rust. Full nil safety is impossible without language support, but we can catch common patterns:

- Unchecked type assertions: `x := val.(Type)` without `, ok`
- Nil pointer dereference after error: `if err != nil { log } // continues to use val`
- Pointer fields without nil checks before use

Recommendation: integrate `nilaway` (Uber's tool) via Phase 5 external integration rather than reimplementing.

---

## Phase 5 — External tool integration

**Effort**: ~1 day.
**Closes**: ~10 additional rules via existing tools.

Add a `external` section to `.goarch.yml` that runs third-party tools as part of the goarch validation pipeline:

```yaml
external:
  - name: "golangci-lint"
    cmd: "golangci-lint run ./..."
    # Covers: naming conventions (revive), staticcheck (~400 checks),
    # govet, errcheck, ineffassign, unused

  - name: "nilaway"
    cmd: "nilaway ./..."
    # Covers: nil safety (S25-S26)

  - name: "dupl"
    cmd: "jscpd --min-tokens 50 --format go ."
    # Covers: copy-paste detection (PMD CPD)
```

Implementation in `cmd/goarch/main.go`: before running go vet, iterate `external` commands and run each. Any failure blocks the build.

This turns goarch into a single entry point for ALL code quality checks — own analyzers + third-party tools.

---

## Coverage Matrix

Final state after all phases:

| Spec Section | Rules | Phase | Analyzer |
|-------------|-------|-------|----------|
| Layer isolation (S1-S10, L1-L7) | 20 | Done | `layerguard` |
| Circular deps (S24/L13) | 1 | Free | Go compiler |
| Annotation placement (S9, S11) | 2 | Free | Go structural (no annotations) |
| Security annotations (S10) | 1 | Phase 3 | `authguard` |
| ProcessBuilder isolation (S12) | 1 | Phase 1 | `execguard` |
| MapStruct (S13) | 1 | Free | N/A in Go |
| DTOs are records (S14) | 1 | Free | Go structs |
| JSON naming (S15-S16, L8) | 3 | Phase 3 | `tagguard` |
| Sensitive fields (S17) | 1 | Done | `secretguard` |
| API type leaks (S18) | 1 | Phase 1 | `apileak` |
| Error placement (S19, L17) | 2 | Phase 3 | `errguard` |
| No @Transactional (S20) | 2 | Free | Go structural |
| No field injection (S21) | 1 | Free | Go structural |
| No direct Redis (S22) | 1 | Phase 1 | `execguard` |
| No direct env (S23, L12) | 2 | Done | `execguard` |
| No null (S25-S26, L14-L15) | 6 | Phase 5 | `nilaway` external |
| Naming conventions (S27, L16) | 2 | Phase 5 | `golangci-lint` external |
| Method length | 1 | Phase 2 | `funlen` |
| Parameter count | 1 | Phase 2 | `argcount` |
| Cyclomatic complexity | 1 | Phase 2 | `complexity` |
| Fan-out complexity | 1 | Done | `fanout` |
| Method count | 1 | Done | `methodcount` |
| No system out (L11) | 2 | Phase 1 | `execguard` |
| No Thread.sleep (L10) | 1 | Phase 1 | `execguard` |
| Formatting | 1 | Free | `gofmt` |
| Banned libraries | 18 | Phase 1 | `execguard` |
| Banned libraries (Java-specific) | 10 | Free | N/A in Go |
| Copy-paste detection | 2 | Phase 5 | `jscpd` external |
| Error Prone equiv | 1 | Phase 5 | `staticcheck` external |
| Dependency governance | 1 | Phase 2 | `depban` |
| **Total** | **84** | | **84/84** |

# ArchUnit & Build Toolchain Audit: Java vs Rust vs Go

**Date:** 2026-03-05
**Source:** `~/Projects/schedulr` — 74 ArchUnit rules across 3 test files + 6 build plugins
**Purpose:** Per-rule analysis of what each guardrail does, why it exists, and how to replicate it in Rust and Go

---

## Table of Contents

1. [Layer Isolation Rules](#1-layer-isolation-rules-20-rules)
2. [Annotation & Placement Rules](#2-annotation--placement-rules-10-rules)
3. [Type Constraint Rules](#3-type-constraint-rules-5-rules)
4. [No-Null Rules](#4-no-null-rules-6-rules)
5. [Naming & Style Rules](#5-naming--style-rules-7-rules)
6. [Project-Wide Bans](#6-project-wide-bans-10-rules)
7. [Banned Library Rules](#7-banned-library-rules-28-rules)
8. [Build Toolchain](#8-build-toolchain-6-tools)
9. [Summary Scorecard](#9-summary-scorecard)

---

## 1. Layer Isolation Rules (20 rules)

These rules enforce that each package/module can only depend on a specific allowlist of other packages. This prevents spaghetti dependencies, keeps modules independently testable, and ensures changes in one layer don't cascade unpredictably.

### S1: `subprocess_allowed_deps`
**What:** `com.schedulr.subprocess` can only depend on `subprocess`, `exception`, and `java.*`.
**Why:** Subprocess is the dangerous layer — it runs shell commands via `ProcessBuilder`. It must be completely isolated from business logic, frameworks, and transports. If an agent modifies subprocess code, the blast radius is contained.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Separate `subprocess` crate in workspace. `Cargo.toml` lists only `graf-error` (exception equivalent) and std. The compiler refuses any other import. **Compile-time enforced.** |
| **Go** | Separate `internal/subprocess` package. Go allows importing anything visible, so this is convention-only. Could use `depguard` linter with allowlist config. **Lint-time enforced.** |

### S2: `taskdefinition_allowed_deps`
**What:** `com.schedulr.taskdefinition` can only depend on `taskdefinition`, `exception`, limitr config types, and `java.*`.
**Why:** Task definitions are pure data — they describe what to execute without knowing how. No framework deps, no I/O. This ensures task definitions are portable across transports (CLI, REST, MCP).

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Separate `task-definition` crate. Dependencies: `graf-error`, `serde` (for serialization), std only. **Compile-time enforced.** |
| **Go** | Separate `taskdef` package with `depguard` allowlist. **Lint-time enforced.** |

### S3: `api_allowed_deps`
**What:** `com.schedulr.api` can only depend on `api`, `limitr.json` marker, and `java.*`.
**Why:** The API layer defines the contract between core logic and transports. It must be framework-free so any transport (CLI, REST, MCP, future gRPC) can use it without pulling in framework-specific code.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `api` crate with zero framework deps. Only depends on `serde` and std. **Compile-time enforced.** |
| **Go** | `api` package with `depguard`. **Lint-time enforced.** |

### S4: `rest_transport_allowed_deps`
**What:** `com.schedulr.transport.rest` can depend on `transport.rest`, `api`, `json`, `exception`, Spring, MapStruct, Jakarta, `java.*`.
**Why:** REST transport may use Spring/Jakarta for HTTP handling, but cannot reach into core, subprocess, or other transports. Each transport is isolated.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `transport-rest` crate depending on `api` crate + `axum`/`actix`. Cannot import `subprocess` or `transport-cli` because they're not in its `Cargo.toml`. **Compile-time enforced.** |
| **Go** | `transport/rest` package with `depguard` allowing `api` + `net/http`. **Lint-time enforced.** |

### S5: `mcp_transport_allowed_deps`
**What:** `com.schedulr.transport.mcp` can depend on `transport.mcp`, `api`, MCP SDK, Spring, Jakarta, `java.*`.
**Why:** Same isolation as REST but for the MCP (Model Context Protocol) transport.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `transport-mcp` crate depending on `api` + MCP SDK crate. **Compile-time enforced.** |
| **Go** | `transport/mcp` package with `depguard`. **Lint-time enforced.** |

### S6: `cli_transport_allowed_deps`
**What:** `com.schedulr.transport.cli` can depend on `transport.cli`, `api`, Spring, Picocli, Jakarta, `java.*`.
**Why:** CLI transport uses Picocli for argument parsing but cannot reach into other transports or core internals.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `transport-cli` crate depending on `api` + `clap`. **Compile-time enforced.** |
| **Go** | `transport/cli` package with `depguard` allowing `api` + `cobra`/`pflag`. **Lint-time enforced.** |

### S7: `extensions_allowed_deps`
**What:** `com.schedulr.extensions` can depend on `extensions`, `api`, and `java.*` only.
**Why:** Extensions are pluggable add-ons. They must go through the API layer, never reach into internals.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `extensions` crate depending on `api` only. **Compile-time enforced.** |
| **Go** | `extensions` package with `depguard`. **Lint-time enforced.** |

### S8: `config_allowed_deps`
**What:** `com.schedulr.config` can depend on `config`, `api`, Spring Boot, `java.*`.
**Why:** Config reads application properties. It depends on Spring Boot for `@ConfigurationProperties` but nothing else.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Config module within the binary crate, using `serde` + `config` crate. No cross-crate leakage. **Compile-time enforced via crate boundaries.** |
| **Go** | `config` package with `depguard`. **Lint-time enforced.** |

### S-parser: `parser_allowed_deps`
**What:** `com.schedulr.parser` can depend on `parser`, `taskdefinition`, `exception`, limitr types, Jackson, `java.*`.
**Why:** The parser converts YAML into task definitions. It uses Jackson for YAML parsing but cannot depend on the execution engine, transports, or subprocess.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `parser` crate depending on `task-definition` + `serde_yaml`. **Compile-time enforced.** |
| **Go** | `parser` package with `depguard` allowing `taskdef` + `gopkg.in/yaml.v3`. **Lint-time enforced.** |

### S-core: `core_allowed_deps`
**What:** `com.schedulr.core` can depend on `core`, `api`, `subprocess`, `taskdefinition`, `parser`, `config`, `exception`, limitr types, Spring, SLF4J, `java.*`. Explicitly cannot depend on any transport.
**Why:** Core is the integration layer — it wires subprocess execution to task definitions with rate limiting. No transport/presentation code allowed.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `core` crate with deps on `api`, `subprocess`, `task-definition`, `parser`, `config`. Transport crates depend on `core`, not the reverse. **Compile-time enforced — Rust's crate DAG prevents reverse deps.** |
| **Go** | `core` package with `depguard` excluding `transport/*`. **Lint-time enforced.** |

### S24 / L13: `no_circular_dependencies`
**What:** No circular dependencies between top-level packages within each module.
**Why:** Cycles make code impossible to reason about in isolation. If A depends on B depends on A, changing either affects both.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **Compile error.** Cargo does not allow circular crate dependencies. Period. Within a single crate, module cycles are also forbidden by the compiler. **Impossible to violate.** |
| **Go** | **Compile error** for package-level cycles. Go forbids circular imports. **Impossible to violate.** |

### L1: `no_framework_dependencies` (limitr)
**What:** The entire `com.limitr` package has zero framework dependencies — no Spring, no Quarkus, no Jakarta Enterprise.
**Why:** Limitr is a pure library. It must work in any Java context, not just Spring Boot. Framework coupling would make it unusable outside schedulr.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Crate with no framework deps in `Cargo.toml`. Only `serde`, std. **Compile-time enforced.** |
| **Go** | `depguard` ban on framework packages. **Lint-time enforced.** |

### L2-L6: limitr internal layer isolation (5 rules)
**What:** Each limitr sublayer (statebackend, ratelimiter, circuitbreaker, retry, executor) has a strict dependency allowlist. The executor depends on the others via interfaces, never on implementations.
**Why:** The limitr library is internally layered. Each resilience pattern (rate limit, circuit break, retry, bulkhead) is independent and composable. The executor orchestrates them without knowing their implementation details.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Each pattern as a separate module or sub-crate within a workspace. Traits define boundaries. The executor module depends on trait definitions, not concrete types. **Compile-time enforced via trait bounds.** |
| **Go** | Interfaces in an `api` package, implementations in separate packages. `depguard` for enforcement. **Lint-time for import rules, compile-time for interface satisfaction.** |

### L7: Executor depends on interfaces only (3 rules)
**What:** `com.limitr.executor` cannot depend on any class containing "InMemory", "Redis", or "Default" in its name.
**Why:** The executor must use dependency injection — it programs against interfaces, not implementations. This ensures implementations (in-memory for testing, Redis for production) are pluggable without modifying the executor.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Executor takes `impl RateLimiter`, `impl CircuitBreaker`, etc. as generic parameters or trait objects. The concrete types live in separate crates that the executor crate doesn't depend on. **Compile-time enforced — if it's not in Cargo.toml, you can't import it.** |
| **Go** | Executor accepts interfaces. Implementations in separate packages. `depguard` prevents importing impl packages. **Lint-time enforced for imports, compile-time for interface compliance.** |

### limitr-redis `allowed_deps`
**What:** `com.limitr.redis` can depend on limitr core interfaces, Lettuce (Redis client), and `java.*` only.
**Why:** The Redis implementation of limitr's state backend. It depends on the limitr interfaces (not implementations) and the Redis client, nothing else.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `limitr-redis` crate depending on `limitr-api` (traits) + `redis` crate. **Compile-time enforced.** |
| **Go** | `limitr/redis` package with `depguard`. **Lint-time enforced.** |

---

## 2. Annotation & Placement Rules (10 rules)

These rules enforce that certain framework constructs are used only in the correct locations.

### S9: `rest_controllers_only_in_transport_rest`
**What:** Classes annotated with `@RestController` must live in `com.schedulr.transport.rest`.
**Why:** HTTP endpoint definitions must be in the REST transport layer. If a controller appears elsewhere, the layer isolation is broken.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Route handlers are functions registered with the router in one place. No annotation magic — you explicitly call `router.route("/path", handler)`. **Structural — the pattern doesn't exist to violate.** |
| **Go** | Same — `http.HandleFunc` or router registration in one file. No magic annotations. **Structural.** |

### S10: `rest_methods_must_have_security_annotation`
**What:** Every public method in a `@RestController` must have `@PreAuthorize`, `@Secured`, or `@PermitAll`.
**Why:** Prevents accidentally exposing an unauthenticated endpoint. Every endpoint must have an explicit security decision, even if that decision is "permit all."

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Auth middleware or extractor pattern. In axum: `async fn handler(auth: AuthGuard, ...)` — if `AuthGuard` is a required extractor, the request fails without auth. For "permit all" endpoints, use a different router group without the middleware. **Compile-time enforced — missing extractor means missing parameter, function signature doesn't match.** |
| **Go** | Middleware chain pattern. Auth middleware wraps handlers. Explicit `publicRouter` vs `authedRouter`. **Convention-based — no compile-time guarantee.** Custom linter could check that all handlers in `authedRouter` use auth middleware. |

### S11: `configuration_properties_in_config_packages`
**What:** `@ConfigurationProperties` classes must live in `..config..` packages.
**Why:** Config binding should happen in one place, not scattered across the codebase.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Config structs live in a `config` module. `serde::Deserialize` on config structs. No framework magic that could be misplaced. **Structural — config deserialization happens in one place by design.** |
| **Go** | Config structs in `config` package. `envconfig` or `viper` used in one place. **Convention-based.** |

### S12: `process_builder_only_in_subprocess`
**What:** No class outside `com.schedulr.subprocess` may use `ProcessBuilder`.
**Why:** Shell execution is dangerous. Containing it in one package means security review, input sanitization, and sandboxing logic all live in one place. Agents can't accidentally spawn processes from a REST handler.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `std::process::Command` only used in the `subprocess` crate. Other crates don't import it because it's a std type — **you'd need a lint or `clippy` deny to enforce this**. Alternative: wrap `Command` in a `subprocess` crate and use `#![deny(clippy::disallowed_methods)]` with `std::process::Command` in the disallowed list for all other crates. **Lint-time enforced via clippy config.** |
| **Go** | `os/exec` only in `subprocess` package. `depguard` or `revive` linter rule to ban `os/exec` in other packages. **Lint-time enforced.** |

### S13: `mapstruct_mappers_must_be_abstract_classes`
**What:** MapStruct mappers must be abstract classes, not interfaces.
**Why:** Spring+MapStruct quirk — abstract class mappers work better with GraalVM native image than interface mappers.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **N/A.** No code generation for mapping. Rust uses `From`/`Into` trait implementations — they're explicit, type-checked, and have zero codegen. |
| **Go** | **N/A.** Go uses explicit conversion functions. No codegen mappers. |

### S17: `sensitive_fields_must_be_annotated`
**What:** Fields named `password`, `token`, `secret`, `ssn`, `creditCard`, `cvv`, `apiKey`, `privateKey` must have a `@Sensitive` annotation.
**Why:** The `@Sensitive` annotation triggers log redaction and serialization masking. Without it, secrets leak into logs and JSON responses.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Use a `Secret<T>` wrapper type (from the `secrecy` crate). `Secret<String>` implements `Debug` as `Secret([REDACTED])` and does NOT implement `Display` or `Serialize` by default. You must explicitly call `.expose_secret()` to access the value. **Compile-time enforced — you literally cannot print or serialize a secret without going through the expose method.** Stronger than the Java approach because the annotation can be forgotten; the wrapper type cannot be circumvented. |
| **Go** | Custom `Secret` type that implements `fmt.Stringer` returning `[REDACTED]` and `json.Marshaler` returning `"***"`. **Runtime enforced — relies on using the type, which is convention.** Could use `go vet` custom analyzer. |

### S18: `api_methods_must_not_return_limitr_internals`
**What:** Public methods in `com.schedulr.api` must not return types from `com.limitr.executor` or `com.limitr.statebackend`.
**Why:** The API layer is the public contract. Leaking internal executor/statebackend types into the API creates tight coupling — callers would need to know limitr internals.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | If `limitr-executor` and `limitr-statebackend` types are `pub(crate)` or not re-exported from the API crate, they simply can't appear in the API's public signatures. **Compile-time enforced via visibility.** |
| **Go** | Internal types in `internal/` directory are already unexportable. For non-internal packages, this is convention. **Partially compile-time (for `internal/`), otherwise convention.** |

### S19 / L17: `exceptions_in_exception_package`
**What:** All exception classes must live in `com.schedulr.exception` (or `com.limitr.exception`).
**Why:** Centralizes error types. Makes it easy to audit all possible error conditions. Prevents ad-hoc exceptions scattered across the codebase.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Define all error types in an `error` module or `error` crate. Use `thiserror` for derive macros. Other modules return `Result<T, GrafError>`. **Convention-based — the compiler doesn't prevent defining error types elsewhere, but the pattern of returning `Result<T, CentralError>` naturally centralizes errors.** Could lint with a custom clippy rule. |
| **Go** | Errors in an `errors` package. Custom `go vet` analyzer or `revive` rule to check. **Convention-based.** |

### S20: `no_jakarta_transactional` / `no_spring_transactional`
**What:** Neither `@Transactional` annotation (Jakarta or Spring) may be used anywhere.
**Why:** Transaction boundaries must be explicit, not annotation-driven. Annotation-based transactions have subtle semantics (proxy-based, self-call issues, propagation defaults) that agents get wrong. Explicit `connection.transaction(|txn| { ... })` is clearer and harder to misuse.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **N/A — the pattern doesn't exist.** Rust has no annotation-driven transactions. You call `conn.transaction(|tx| { ... })` explicitly. Transaction boundaries are visible in the code. **Structural — impossible to violate.** |
| **Go** | Same — `tx, err := db.Begin()` is explicit. No annotation magic. **Structural.** |

### S21: `no_field_injection`
**What:** No `@Autowired` on fields. Constructor injection only.
**Why:** Field injection hides dependencies, makes testing harder, and allows circular dependencies that constructor injection would catch. Constructor injection makes all dependencies explicit in the constructor signature.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **N/A — no DI framework.** Dependencies are struct fields set in the constructor (`new()`). There's no way to inject a field without passing it explicitly. **Structural — impossible to violate.** |
| **Go** | Same — struct fields set in constructor functions. No DI framework magic. **Structural.** |

---

## 3. Type Constraint Rules (5 rules)

### S14: `dtos_must_be_records`
**What:** All classes in `..dto..` packages must be Java records.
**Why:** DTOs are pure data carriers. Records are immutable, have auto-generated `equals`/`hashCode`/`toString`, and have a compact syntax. Prevents agents from making DTOs mutable classes with setters.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Structs are immutable by default. `#[derive(Debug, Clone, PartialEq, serde::Serialize, serde::Deserialize)]` gives you the record equivalent. Mutability requires explicit `mut`. **Language default.** |
| **Go** | Structs are mutable by default. No language-level enforcement of immutability. Convention to use value receivers and no setter methods. **Convention-based only.** |

### S15: `rest_dtos_must_implement_snake_case_dto`
**What:** REST DTO records must implement `SnakeCaseDto` marker interface.
**Why:** REST API uses `snake_case` JSON (standard for REST). The marker interface triggers Jackson's snake_case naming strategy. Without it, Java's `camelCase` field names leak into the REST API.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `#[serde(rename_all = "snake_case")]` on the struct. Per-struct, explicit, checked at compile time. If you forget it, the fields serialize in Rust's native `snake_case` anyway (Rust convention matches REST convention). **Built into the language convention + serde.** |
| **Go** | `json:"field_name"` struct tags. Must be applied per-field. No global enforcement — linters like `govet` check for missing tags but not naming convention. **Partially enforced by `govet`.** |

### S16 / L8: `api_records_implement_camel_case_json`
**What:** Public records in the API package must implement `CamelCaseJson` marker interface.
**Why:** Internal API uses `camelCase` (Java convention). The marker ensures consistent serialization.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `#[serde(rename_all = "camelCase")]` on API structs. **Explicit per-struct via serde attribute.** |
| **Go** | `json:"fieldName"` struct tags. **Manual per-field.** |

### L9: `statebackend_impls_are_final`
**What:** All non-interface classes in `com.limitr.statebackend` must be `final`.
**Why:** State backend implementations should not be subclassed. They are concrete, sealed implementations of the state backend interface.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **All structs are "final" by default.** Rust has no inheritance. You cannot subclass a struct. **Language guarantee — impossible to violate.** |
| **Go** | Go has no inheritance either. Structs cannot be subclassed. Embedding is composition, not inheritance. **Language guarantee.** |

---

## 4. No-Null Rules (6 rules)

### S25 / L14: `no_optional_fields`
**What:** No field may have type `java.util.Optional`.
**Why:** `Optional` was designed for return types, not fields. Optional fields are not serializable by default, waste memory (object wrapper), and create ambiguity — is a missing field null or `Optional.empty()`?

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `Option<T>` IS the standard for optional values and is a first-class type. The compiler forces exhaustive handling (`match`, `if let`, `unwrap_or`). Unlike Java's `Optional`, Rust's `Option` is zero-cost, serializable, and the idiomatic way to represent absence. **The rule doesn't apply — Option<T> is correct and safe in Rust.** |
| **Go** | Pointers (`*T`) for optional fields. `nil` checks required. No compile-time enforcement of nil handling. **Weaker than both Java and Rust.** |

### S26 / L15: `no_nullable_annotations` (6 instances across 3 test files)
**What:** No `@Nullable` annotation from any vendor (Jakarta, Spring, javax).
**Why:** If you ban `Optional` fields AND ban `@Nullable`, you're saying: every field must be non-null, always. This forces the code to handle absence at the boundary (constructor/factory) rather than throughout the codebase.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **There is no null.** A `String` field always contains a string. An `Option<String>` field explicitly declares optionality and the compiler forces you to handle it. This is strictly superior — you get the "no null in fields" guarantee plus safe optionality where needed. **Language guarantee.** |
| **Go** | Pointers are nullable. Slices, maps, interfaces, channels are all nillable. No compile-time null safety. `nilaway` linter helps but is not complete. **Weak — Go's biggest gap vs Java+ArchUnit and Rust.** |

---

## 5. Naming & Style Rules (7 rules)

### S27 / L16: `no_snake_case_field_names`
**What:** No non-static field may contain an underscore.
**Why:** Java convention is `camelCase`. Snake case fields in Java usually indicate copy-paste from JSON/SQL or confused convention.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Rust convention IS `snake_case`. `clippy::non_snake_case` warns on `camelCase` fields. **Inverted rule, same enforcement — clippy catches convention violations.** |
| **Go** | Go convention is `camelCase` for private, `PascalCase` for exported. `golint`/`revive` catch violations. **Lint-time enforced.** |

### Spotless + Google Java Format
**What:** All Java source is auto-formatted with Google Java Format.
**Why:** Eliminates formatting debates. Every file looks the same. Agents can't introduce inconsistent formatting.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `rustfmt` — built into the toolchain. `cargo fmt` auto-formats. `cargo fmt --check` fails CI. Zero configuration needed (standard Rust style). **Built-in, zero config.** |
| **Go** | `gofmt` — built into the toolchain. Same pattern. **Built-in, zero config.** |

### Checkstyle: `MethodLength` (max 75 lines)
**What:** No method may exceed 75 lines.
**Why:** Long methods are hard to understand and review. Agents tend to generate large methods; this forces decomposition.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `clippy::too_many_lines` — configurable via `clippy.toml`: `too-many-lines-threshold = 75`. **Lint-time enforced.** |
| **Go** | `funlen` linter via `golangci-lint`: `funlen: lines: 75`. **Lint-time enforced.** |

### Checkstyle: `ParameterNumber` (max 7)
**What:** No method may have more than 7 parameters.
**Why:** Too many params indicate the method is doing too much or needs a parameter object.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `clippy::too_many_arguments` — default is 7. Exact match. **Lint-time enforced (default clippy).** |
| **Go** | No built-in linter for this. Could use `revive` with `argument-limit` rule. **Lint-time enforced via third-party.** |

### Checkstyle: `CyclomaticComplexity` (max 15)
**What:** No method may have cyclomatic complexity above 15.
**Why:** High complexity = many branches = hard to test and reason about. Agents generate complex conditionals; this forces simplification.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `clippy::cognitive_complexity` — configurable via `clippy.toml`: `cognitive-complexity-threshold = 15`. Note: clippy uses "cognitive complexity" (slightly different metric) but serves the same purpose. **Lint-time enforced.** |
| **Go** | `gocyclo` or `cyclop` via `golangci-lint`. Configurable threshold. **Lint-time enforced.** |

### Checkstyle: `ClassFanOutComplexity` (max 20)
**What:** No class may depend on more than 20 other classes.
**Why:** High fan-out means a class is coupled to too many things. Changes in any of those 20 classes could break this one.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **No direct equivalent in clippy.** Crate boundaries naturally limit fan-out (a crate only sees its declared dependencies). For intra-crate fan-out, no standard tool. **Gap — requires custom lint or discipline.** |
| **Go** | **No direct equivalent.** `gocritic` has some coupling metrics but not a fan-out limit. **Gap.** |

### Checkstyle: `MethodCount` (max 25 public methods)
**What:** No class may have more than 25 public methods.
**Why:** A class with 25+ public methods is doing too much. Split it.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **No direct equivalent in clippy.** Rust's `impl` blocks can be split across files, which naturally distributes methods. Traits also cap interface size. **Gap — requires custom lint or discipline.** |
| **Go** | **No direct equivalent.** Could use a custom linter. Go interfaces are naturally small (convention). **Gap.** |

---

## 6. Project-Wide Bans (10 rules)

### S22: `no_direct_redis_usage`
**What:** No schedulr code may use Lettuce or Spring Data Redis directly.
**Why:** Redis access goes through limitr's abstraction. Schedulr shouldn't know about the storage backend.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Don't add `redis` crate to the main crate's `Cargo.toml`. **Compile-time enforced — no dep = no import possible.** |
| **Go** | `depguard` ban on `github.com/redis/go-redis`. **Lint-time enforced.** |

### S23 / L12: `no_system_getenv`
**What:** No direct `System.getenv()` calls.
**Why:** Environment variables should be read through the config layer. Direct `getenv` calls scatter configuration access, make testing harder, and hide dependencies on environment state.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `clippy::disallowed_methods` with `std::env::var` and `std::env::var_os` in the disallowed list. **Lint-time enforced via clippy config.** |
| **Go** | `depguard` or custom `go vet` analyzer banning direct `os.Getenv`. **Lint-time enforced.** |

### L10: `no_thread_sleep_outside_retry`
**What:** `Thread.sleep()` may only be called from the retry package.
**Why:** Sleep indicates a polling or waiting pattern. These should only exist in the retry logic, not scattered through business code.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `clippy::disallowed_methods` with `std::thread::sleep` banned globally, plus an `#[allow(clippy::disallowed_methods)]` in the retry module only. **Lint-time enforced with targeted exception.** |
| **Go** | `depguard` or `revive` rule banning `time.Sleep` with package exception. **Lint-time enforced.** |

### L11: `no_system_out` / `no_system_err`
**What:** No `System.out` or `System.err` access.
**Why:** All output must go through the logging framework (SLF4J). Direct stdout/stderr bypasses log levels, formatting, and log aggregation.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `clippy::print_stdout` and `clippy::print_stderr` — **built-in clippy lints, just enable them.** `#[deny(clippy::print_stdout, clippy::print_stderr)]` in `lib.rs`. For the CLI binary crate (which needs stdout), allow it only there. **Lint-time enforced, built-in.** |
| **Go** | `forbidigo` linter via `golangci-lint` — bans `fmt.Print*` patterns. **Lint-time enforced.** |

### S10 (revisited as ban): No unauthenticated endpoints
Covered in section 2 above.

---

## 7. Banned Library Rules (28 rules)

These rules exist at two levels: ArchUnit (class-level import checks) and Maven Enforcer (dependency-level artifact checks). In Rust/Go, both collapse into one mechanism.

### Banned JSON libraries (gson, org.json, fastjson, fastjson2, codehaus jackson)
**Why:** Project standardizes on Jackson (Fasterxml). Multiple JSON libraries create inconsistent serialization behavior.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Standardize on `serde` + `serde_json`. Ban alternatives via `cargo-deny`: `[bans] deny = [{name = "json"}, {name = "simd-json"}, ...]`. **Build-time enforced.** |
| **Go** | Standardize on `encoding/json` or `json-iterator`. Ban alternatives via `depguard`. **Lint-time enforced.** |

### Banned logging libraries (JUL, log4j, commons-logging)
**Why:** SLF4J is the standard facade. Multiple logging frameworks cause classpath conflicts and log loss.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Standardize on `tracing` (or `log`). Ban alternatives via `cargo-deny`. **Build-time enforced.** |
| **Go** | Standardize on `slog` (stdlib structured logging). Ban `logrus`, `zap` etc. via `depguard`. **Lint-time enforced.** |

### Banned datetime libraries (joda-time, threetenbp)
**Why:** Java 8+ has `java.time`. Legacy datetime libraries are unnecessary.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Standardize on `std::time` + `time` crate (or `chrono`, pick one). Ban the other via `cargo-deny`. **Build-time enforced.** |
| **Go** | `time` stdlib. No alternatives needed. **N/A — stdlib is sufficient.** |

### Banned javax namespace (servlet, persistence, validation, inject, xml.bind)
**Why:** Jakarta namespace replaced javax. These are legacy.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **N/A — Java-specific concern.** |
| **Go** | **N/A.** |

### Banned HTTP client libraries (Apache HTTP, commons-httpclient)
**Why:** Standardize on one HTTP client (Spring's RestClient or Java HttpClient).

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Standardize on `reqwest`. Ban `hyper` (direct), `ureq`, `surf` etc. via `cargo-deny`. **Build-time enforced.** |
| **Go** | `net/http` stdlib. Ban `resty`, `gentleman` via `depguard`. **Lint-time enforced.** |

### Banned utility libraries (commons-io, commons-lang, commons-collections, commons-beanutils)
**Why:** Modern Java stdlib covers these. Extra dependencies increase binary size and attack surface.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Std is rich. Be selective with deps. `cargo-deny` can limit total dep count or ban specific crates. **Build-time enforced.** |
| **Go** | Stdlib is comprehensive. `depguard` for specific bans. **Lint-time enforced.** |

### Banned Guava
**Why:** Guava is huge and most of its functionality is in modern Java stdlib. Also conflicts with GraalVM native image in some cases.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **N/A — no equivalent monolith library.** Rust ecosystem is granular (small, focused crates). |
| **Go** | **N/A.** |

### Banned CGLib
**Why:** Runtime bytecode generation. Incompatible with GraalVM native image without extra configuration.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **N/A — no runtime code generation in Rust.** Macros are compile-time. |
| **Go** | No runtime code generation either. **N/A.** |

### Banned testing frameworks (junit4, powermock, testng, easymock)
**Why:** Standardize on JUnit 5 + Mockito. Legacy test frameworks cause classpath conflicts.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Built-in `#[test]` + `#[cfg(test)]`. No framework needed. Can ban test framework crates via `cargo-deny` if desired. **Structural — the language has built-in testing.** |
| **Go** | Built-in `testing` package. `depguard` to ban `testify` if desired. **Structural.** |

### Banned Quarkus / SmallRye / MicroProfile
**Why:** Project uses Spring Boot. Alternative frameworks must not leak in.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Pick one web framework (axum, actix). Ban others via `cargo-deny`. **Build-time enforced.** |
| **Go** | Pick one router. Ban others via `depguard`. **Lint-time enforced.** |

### Banned Caffeine
**Why:** Caffeine's internals break GraalVM native image compilation.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **N/A — no equivalent GraalVM issue.** Rust compiles natively. All caches work at compile time. |
| **Go** | **N/A.** |

---

## 8. Build Toolchain (6 tools)

### Error Prone (compile-time bug detection, ~400 checks)
**What:** Catches null dereference, resource leaks, concurrency bugs, dead code, API misuse at compile time.
**Why:** Java's type system is too weak to catch these bugs. Error Prone adds the missing checks.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **The compiler catches all of these by default.** Null dereference: impossible (no null). Resource leaks: ownership + RAII (Drop). Concurrency bugs: Send/Sync traits + borrow checker. Use-after-free: compile error. Data races: compile error. Dead code: `#[warn(dead_code)]` default. **Strictly superior — these are compile errors, not warnings.** |
| **Go** | `go vet` catches some (printf misuse, unreachable code). `staticcheck` catches more (400+ checks, closest Go equivalent to Error Prone). `nilaway` for null safety. **Lint-time — weaker than both Java+ErrorProne and Rust.** |

### Maven Enforcer (dependency governance)
**What:** Bans duplicate classes on classpath, enforces dependency upper bounds, blocks banned artifacts.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | `cargo-deny` — handles bans, license checks, advisory DB checks. Cargo itself handles version resolution (no "duplicate classes" issue — Rust allows multiple versions of the same crate but they don't conflict). **Build-time enforced.** |
| **Go** | `go mod tidy` + `depguard`. Go modules handle version resolution cleanly. **Build-time + lint-time.** |

### Checkstyle (code metrics)
Covered per-rule in section 5 above.

### Spotless (auto-formatting)
Covered in section 5 above.

### PMD CPD (copy-paste detection, 50 token minimum)
**What:** Fails the build if any code block of 50+ tokens is duplicated.
**Why:** Agents frequently duplicate code instead of extracting shared logic. CPD catches this.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | **No mature Rust-native CPD tool.** `jscpd` supports Rust via its language-agnostic tokenizer. Run `jscpd --min-tokens 50 --format rust`. **External tool, not build-integrated.** |
| **Go** | `jscpd` with `--format go`. Or `dupl` linter via `golangci-lint`. **Lint-time enforced (dupl is Go-native).** |

### jscpd (additional copy-paste detection)
**What:** Same as CPD but runs as a standalone Node.js tool. Configured in `build/.jscpd.json` with 50 token threshold.
**Why:** Belt-and-suspenders with Maven CPD. Also supports non-Java files.

| Language | How to replicate |
|----------|-----------------|
| **Rust** | Same `jscpd` tool works for Rust. Change config: `"format": ["rust"]`. **Works as-is.** |
| **Go** | Same. `"format": ["go"]`. Or use `dupl` (Go-native). **Works as-is.** |

---

## 9. Summary Scorecard

### By rule count

| Category | Total rules | Rust: compile-time | Rust: lint-time | Rust: N/A (not needed) | Rust: gap |
|----------|------------|--------------------|-----------------|-----------------------|-----------|
| Layer isolation | 20 | 18 (crate boundaries) | 0 | 0 | 2 (intra-crate) |
| Annotation/placement | 10 | 4 (structural) | 1 (clippy) | 4 (pattern doesn't exist) | 1 (exception pkg) |
| Type constraints | 5 | 4 (language default) | 1 (serde attr) | 0 | 0 |
| No-null rules | 6 | 6 (language guarantee) | 0 | 0 | 0 |
| Naming/style | 7 | 0 | 5 (rustfmt + clippy) | 0 | 2 (fan-out, method count) |
| Project-wide bans | 10 | 4 (structural) | 5 (clippy::disallowed) | 0 | 1 (exception pkg) |
| Banned libraries | 28 | 0 | 0 | 10 (Java-specific) | 0 |
| Banned libraries (applicable) | 18 | 0 | 18 (cargo-deny) | 0 | 0 |
| **Total** | **84** | **36** | **30** | **14** | **4** |

### Enforcement level comparison

| Enforcement level | Java + ArchUnit | Rust | Go |
|-------------------|----------------|------|-----|
| **Compile-time (impossible to violate)** | 0 rules | 36 rules | 2 rules (cycles only) |
| **Test-time (fails on `mvn test`)** | 74 rules | 0 | 0 |
| **Lint-time (fails on `cargo clippy` / `golangci-lint`)** | 6 tools | 30 rules | ~60 rules |
| **Build-time (fails on `mvn verify` / `cargo deny`)** | 6 tools | 18 rules (cargo-deny) | 18 rules (depguard) |
| **Not needed (language prevents the problem)** | 0 | 14 rules | 4 rules |
| **Gaps (no equivalent)** | 0 | 4 rules | ~10 rules |

### The 4 Rust gaps (what you'd lose)

1. **Intra-crate package allowlists** (2 rules) — Within a single crate, any module can import any other `pub` module. ArchUnit can restrict this; Rust cannot without a custom lint. **Mitigation:** Split into more crates, or accept intra-crate freedom.

2. **Class fan-out complexity** (1 rule) — No clippy equivalent for "this module depends on too many other types." **Mitigation:** Crate boundaries naturally limit this. Monitor manually.

3. **Public method count per type** (1 rule) — No clippy equivalent for "this impl block has too many public methods." **Mitigation:** Trait-based design naturally caps interface size. Monitor manually.

### What Rust gives you that Java + ArchUnit cannot

| Guarantee | Java + ArchUnit | Rust |
|-----------|-----------------|------|
| No null pointer exceptions | Requires discipline + annotations | **Compile-time impossible** |
| No data races | Requires discipline + tools | **Compile-time impossible** |
| No use-after-free | GC handles this (but perf cost) | **Compile-time impossible (zero cost)** |
| No resource leaks | Requires discipline + Error Prone | **Compile-time guaranteed (RAII/Drop)** |
| No inheritance hierarchy bugs | ArchUnit can limit but not prevent | **No inheritance exists** |
| Exhaustive pattern matching | Switch expressions (partial, Java 21+) | **Compile error on non-exhaustive match** |
| Thread safety proof | Annotations + runtime checks | **Compile-time proof (Send/Sync)** |
| Zero-cost abstractions | JIT dependent | **Guaranteed by language spec** |
| 2-5MB binary | 50-80MB native image | **Language default** |
| No runtime reflection | GraalVM config required | **No reflection exists** |

---

## 10. Recommended Rust Toolchain for Equivalent Coverage

```toml
# Cargo.toml workspace — crate boundaries enforce layer isolation

# clippy.toml — configurable lints
too-many-lines-threshold = 75
cognitive-complexity-threshold = 15
too-many-arguments-threshold = 7
disallowed-methods = [
    { path = "std::env::var", reason = "Use config module" },
    { path = "std::env::var_os", reason = "Use config module" },
    { path = "std::process::Command::new", reason = "Use subprocess crate" },
    { path = "std::thread::sleep", reason = "Use retry crate" },
]

# deny.toml — dependency governance (cargo-deny)
[bans]
multiple-versions = "deny"
deny = [
    # Ban alternative JSON libs
    { name = "simd-json" },
    # Ban alternative HTTP clients (if standardizing on reqwest)
    { name = "ureq" },
    { name = "surf" },
    # Ban alternative logging
    { name = "log4rs" },
    # etc.
]

[licenses]
allow = ["MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC"]

[advisories]
db-path = "~/.cargo/advisory-db"
vulnerability = "deny"
unmaintained = "warn"

# .jscpd.json — copy-paste detection
{
    "threshold": 0,
    "minTokens": 50,
    "format": ["rust"],
    "ignore": ["**/target/**", "**/*_test.rs"]
}
```

### Clippy configuration in lib.rs / main.rs:
```rust
#![deny(clippy::print_stdout)]
#![deny(clippy::print_stderr)]
#![deny(clippy::too_many_lines)]
#![deny(clippy::cognitive_complexity)]
#![deny(clippy::too_many_arguments)]
#![deny(clippy::disallowed_methods)]
#![deny(clippy::unwrap_used)]        // force proper error handling
#![deny(clippy::expect_used)]        // force proper error handling
#![deny(clippy::panic)]              // no panics in library code
#![deny(unsafe_code)]                // no unsafe in this project
```

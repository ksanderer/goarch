// Package docs contains microdocs for all goarch rules.
// Each rule has an ID, short description, and detailed explanation
// with fix examples.
package docs

// Rule describes a single architectural rule.
type Rule struct {
	ID      string // e.g. "execguard"
	Name    string // human-readable name
	Short   string // one-line shown alongside violations
	Long    string // detailed explanation with fix examples
}

var rules = map[string]Rule{
	"layerguard": {
		ID:    "layerguard",
		Name:  "Layer Isolation",
		Short: "Enforces dependency allowlists/denylists per package.",
		Long: `RULE: layerguard — Layer Isolation

WHAT IT CHECKS:
  Each package can only import packages from its configured allowlist.
  Imports outside the allowlist are reported as violations.

WHY IT EXISTS:
  Prevents spaghetti dependencies. Keeps modules independently testable.
  Ensures changes in one layer don't cascade unpredictably.
  Example: handler/ should never be imported by provider/ — that would
  create a circular dependency direction.

CONFIGURATION (.goarch.yml):
  rules:
    layerguard:
      layers:
        "internal/domain":
          allow: ["internal/logger"]
          deny_all_others: true        # only allow listed + stdlib
        "internal/provider":
          deny: ["internal/handler/*"]  # block specific imports

FIX:
  1. Move the shared code to a lower-level package that both can import.
  2. Define an interface in the lower-level package, implement in higher.
  3. If the import is intentional, update .goarch.yml to allow it.`,
	},

	"execguard": {
		ID:    "execguard",
		Name:  "Banned Import Guard",
		Short: "Bans specific packages or methods outside allowed locations.",
		Long: `RULE: execguard — Banned Import Guard

WHAT IT CHECKS:
  Certain packages or specific methods from packages are banned outside
  explicitly allowed packages. Two modes:
    - Full package ban: "os/exec" banned everywhere except subprocess/
    - Method-level ban: os.Getenv banned except in config/

WHY IT EXISTS:
  Dangerous or cross-cutting operations should be contained in one place.
  Examples:
    - os/exec in one package = one security review surface
    - os.Getenv in config/ only = all env vars documented in one struct
    - time.Sleep only in retry/ = no accidental polling in business code

CONFIGURATION (.goarch.yml):
  rules:
    execguard:
      banned:
        - pkg: "os/exec"
          except: ["internal/subprocess"]
          reason: "Shell execution only in subprocess"
        - pkg: "os"
          methods: ["Getenv", "Setenv"]
          except: ["internal/config"]
          reason: "Use config package for env vars"

FIX:
  1. Move the call to the allowed package and expose it via a function.
  2. If the ban is too strict, add your package to the 'except' list.

  Example — before (violation):
    // internal/handler/completions.go
    telemetry := os.Getenv("TELEMETRY") == "true"

  Example — after (fixed):
    // internal/config/config.go
    type Config struct {
        Telemetry bool
    }
    // internal/handler/completions.go
    telemetry := cfg.Telemetry`,
	},

	"secretguard": {
		ID:    "secretguard",
		Name:  "Sensitive Field Guard",
		Short: "Ensures sensitive fields use a Secret wrapper type.",
		Long: `RULE: secretguard — Sensitive Field Guard

WHAT IT CHECKS:
  Struct fields with names matching sensitive patterns (apikey, secret,
  password, etc.) must use a designated wrapper type instead of plain
  string. Matching is word-boundary aware:
    "apikey" matches OpenRouterAPIKey ✓
    "token" does NOT match PromptTokens ✗ (different word boundary)

WHY IT EXISTS:
  Plain string secrets can accidentally leak into:
    - Logs: logger.Info().Interface("cfg", cfg) → all keys visible
    - Error messages: fmt.Errorf("failed with cfg: %v", cfg)
    - Debug output: fmt.Println(cfg) → all fields visible

  A Secret wrapper type redacts the value in String()/GoString() output:
    fmt.Println(secret)    → "[REDACTED]"
    secret.Value()         → actual value (explicit opt-in)

DESIGN PRINCIPLE:
  Secret is a scalar type. It protects against accidental leaks in logs
  and error messages by redacting String() and GoString(). It does NOT
  implement MarshalJSON — JSON serialization stays transparent.

  Protection against JSON leaks is handled at the struct level:
    - Use json:"-" on secret fields in API response DTOs
    - Use custom MarshalJSON on the struct when you need control

  This keeps Secret compatible with storage (Redis, DB) where
  json.Marshal must preserve the real value.

CONFIGURATION (.goarch.yml):
  rules:
    secretguard:
      types: ["Secret", "SecretBytes"]     # multiple types allowed
      # type: "Secret"                     # single type also works
      field_patterns: ["apikey", "secret", "password", "accesstoken"]
      except_packages: ["internal/handler"]  # response DTOs are OK

  Pointer types (*Secret, *SecretBytes) are accepted automatically.

FIX:
  1. Create wrapper types in your project:

     type Secret string
     func (s Secret) String() string   { return "[REDACTED]" }
     func (s Secret) GoString() string { return "[REDACTED]" }
     func (s Secret) Value() string    { return string(s) }

     type SecretBytes []byte
     func (s SecretBytes) String() string   { return "[REDACTED]" }
     func (s SecretBytes) GoString() string { return "[REDACTED]" }
     func (s SecretBytes) Value() []byte    { return []byte(s) }

     Do NOT implement MarshalJSON on these types — it breaks storage
     serialization (Redis, caches). Handle JSON redaction at the
     struct level instead.

  2. Change field types:
     // Before
     OpenRouterAPIKey string
     JWTSecret        []byte
     // After
     OpenRouterAPIKey Secret
     JWTSecret        SecretBytes

  3. Use .Value() where the real value is needed:
     req.Header.Set("Authorization", "Bearer " + cfg.APIKey.Value())

  4. For API responses, redact at the struct level:

     // Fields that should never appear in JSON responses:
     type Config struct {
         APIKey Secret ` + "`" + `json:"-"` + "`" + `
     }

     // Auth flows where you need the value in JSON:
     type AuthResponse struct {
         AccessToken  Secret ` + "`" + `json:"-"` + "`" + `
         RefreshToken Secret ` + "`" + `json:"-"` + "`" + `
     }
     func (r AuthResponse) MarshalJSON() ([]byte, error) {
         return json.Marshal(struct {
             AccessToken  string ` + "`" + `json:"access_token"` + "`" + `
             RefreshToken string ` + "`" + `json:"refresh_token"` + "`" + `
         }{
             AccessToken:  r.AccessToken.Value(),
             RefreshToken: r.RefreshToken.Value(),
         })
     }`,
	},

	"fanout": {
		ID:    "fanout",
		Name:  "Fan-Out Complexity",
		Short: "Limits the number of non-stdlib imports per file.",
		Long: `RULE: fanout — Fan-Out Complexity

WHAT IT CHECKS:
  Each file's count of non-stdlib imports must not exceed the configured
  maximum. Stdlib imports (net/http, fmt, etc.) are not counted.

WHY IT EXISTS:
  High fan-out = high coupling. A file importing 20 external packages
  is fragile — changes in any of those packages could break it.
  This rule forces decomposition into smaller, focused files.

CONFIGURATION (.goarch.yml):
  rules:
    fanout:
      max_imports: 15

FIX:
  1. Split the file into multiple files with focused responsibilities.
  2. Extract a helper package that combines related imports.
  3. If the file is a natural integration point (main.go, router setup),
     consider increasing the limit for that specific case.`,
	},

	"methodcount": {
		ID:    "methodcount",
		Name:  "Method Count Limit",
		Short: "Limits the number of exported methods per type.",
		Long: `RULE: methodcount — Method Count Limit

WHAT IT CHECKS:
  Each named type's count of exported (public) methods must not exceed
  the configured maximum.

WHY IT EXISTS:
  A type with 25+ public methods is doing too much — it's a "god object."
  Large interfaces are hard to mock, test, and understand.
  This rule forces splitting into smaller, focused types.

CONFIGURATION (.goarch.yml):
  rules:
    methodcount:
      max_public_methods: 20

FIX:
  1. Split the type into multiple types with distinct responsibilities.
  2. Group related methods behind an interface.
  3. Use composition: embed smaller types into a larger one.`,
	},

	"apileak": {
		ID:    "apileak",
		Name:  "API Type Leak Guard",
		Short: "Prevents internal types from leaking into public API signatures.",
		Long: `RULE: apileak — API Type Leak Guard

WHAT IT CHECKS:
  Exported functions in public API packages must not use types from
  banned internal packages in their parameters or return values.

WHY IT EXISTS:
  If your API package returns an internal executor type, callers are
  coupled to that internal implementation. The API should only expose
  its own types or standard types.

CONFIGURATION (.goarch.yml):
  rules:
    apileak:
      public_packages: ["internal/api"]
      banned_types_in_public:
        - "internal/executor.*"
        - "internal/statebackend.*"

FIX:
  1. Define an interface or DTO in the API package.
  2. Convert the internal type to the API type before returning.
  3. If the type genuinely belongs in the API, move it there.`,
	},

	"funlen": {
		ID:    "funlen",
		Name:  "Function Length Limit",
		Short: "Limits function body length in lines.",
		Long: `RULE: funlen — Function Length Limit

WHAT IT CHECKS:
  Each function's body (from opening { to closing }) must not exceed
  the configured number of lines.

WHY IT EXISTS:
  Long functions are hard to understand, test, and review. They usually
  do too many things and should be decomposed into smaller functions
  with clear names.

CONFIGURATION (.goarch.yml):
  rules:
    funlen:
      max_lines: 75

FIX:
  1. Extract logical blocks into helper functions with descriptive names.
  2. Move data transformations to pure functions.
  3. Use early returns to reduce nesting and length.`,
	},

	"argcount": {
		ID:    "argcount",
		Name:  "Argument Count Limit",
		Short: "Limits the number of function parameters.",
		Long: `RULE: argcount — Argument Count Limit

WHAT IT CHECKS:
  Each function's parameter count (expanding grouped params like
  "a, b int" into 2) must not exceed the configured maximum.

WHY IT EXISTS:
  Functions with many parameters are hard to call correctly and
  indicate the function is doing too much. Use an options struct
  for functions that need many inputs.

CONFIGURATION (.goarch.yml):
  rules:
    argcount:
      max_args: 7

FIX:
  1. Group related parameters into a struct:
     // Before
     func Send(to, from, subject, body string, cc []string, bcc []string, ...) error
     // After
     func Send(msg Message) error

  2. Use functional options pattern for optional parameters.
  3. Split the function if it's doing too many things.`,
	},

	"complexity": {
		ID:    "complexity",
		Name:  "Cyclomatic Complexity",
		Short: "Limits cyclomatic complexity per function.",
		Long: `RULE: complexity — Cyclomatic Complexity

WHAT IT CHECKS:
  Each function's cyclomatic complexity (number of independent paths
  through the code) must not exceed the configured maximum.
  Counted: if, for, range, case, &&, ||, select case. Starts at 1.

WHY IT EXISTS:
  High complexity = hard to test and reason about. A function with
  complexity 20 needs at least 20 test cases for full coverage.

CONFIGURATION (.goarch.yml):
  rules:
    complexity:
      max_complexity: 15

FIX:
  1. Extract conditional branches into named functions.
  2. Replace switch/case with a map lookup.
  3. Use early returns to flatten nested if/else chains.
  4. Split the function into smaller, focused functions.`,
	},

	"depban": {
		ID:    "depban",
		Name:  "Dependency Ban",
		Short: "Bans modules in go.mod and limits dependency count.",
		Long: `RULE: depban — Dependency Ban

WHAT IT CHECKS:
  Scans go.mod for banned module dependencies and optionally limits
  the total number of direct dependencies. Unlike execguard (which
  checks source imports), depban catches modules in go.mod that may
  only be used transitively.

WHY IT EXISTS:
  Dependency governance prevents supply chain risks and bloat.
  Banning specific modules enforces technology standardization
  (e.g. one HTTP router, one logging library).

CONFIGURATION (.goarch.yml):
  rules:
    depban:
      deny:
        - module: "github.com/sirupsen/logrus"
          reason: "Use zerolog"
        - module: "github.com/gin-gonic/gin"
          reason: "Use chi"
      max_dependencies: 50

FIX:
  1. Replace the banned dependency with the approved alternative.
  2. Run 'go mod tidy' to clean up unused dependencies.
  3. If the dependency is needed, discuss with the team and update .goarch.yml.`,
	},

	"tagguard": {
		ID:    "tagguard",
		Name:  "Struct Tag Guard",
		Short: "Validates struct tag naming conventions per package.",
		Long: `RULE: tagguard — Struct Tag Guard

WHAT IT CHECKS:
  Per-package validation of struct field JSON tags:
    - Naming convention: snake_case or camelCase
    - Presence: exported fields must have a json tag (optional)

WHY IT EXISTS:
  Inconsistent JSON serialization causes API compatibility issues.
  REST APIs should use snake_case, internal types may use camelCase.
  Missing tags on exported fields can expose internal names.

CONFIGURATION (.goarch.yml):
  rules:
    tagguard:
      packages:
        "internal/domain":
          json_naming: "snake_case"
          require_json_tags: true
        "internal/api":
          json_naming: "camelCase"

FIX:
  1. Add or fix the json tag:
     // Before
     UserName string
     // After
     UserName string ` + "`" + `json:"user_name"` + "`" + `

  2. Use json:"-" to explicitly omit fields from serialization.`,
	},

	"errguard": {
		ID:    "errguard",
		Name:  "Error Type Placement",
		Short: "Restricts where custom error types can be defined.",
		Long: `RULE: errguard — Error Type Placement

WHAT IT CHECKS:
  Types that implement the error interface (have an Error() string
  method) must be defined in one of the allowed packages.

WHY IT EXISTS:
  Scattered error types make error handling inconsistent. Centralizing
  errors in one package (e.g. internal/domain) makes them discoverable,
  ensures consistent error codes, and simplifies error mapping.

CONFIGURATION (.goarch.yml):
  rules:
    errguard:
      allowed_packages: ["internal/domain"]

FIX:
  1. Move the error type to the allowed package.
  2. If the error is package-specific, use fmt.Errorf or errors.New
     instead of a custom type.
  3. If a new error package is needed, add it to allowed_packages.`,
	},

	"authguard": {
		ID:    "authguard",
		Name:  "Endpoint Auth Coverage",
		Short: "Verifies HTTP routes go through auth middleware.",
		Long: `RULE: authguard — Endpoint Auth Coverage

WHAT IT CHECKS:
  In the router package, HTTP route registrations (r.Get, r.Post, etc.)
  must be inside a Route group that calls Use() with the configured
  auth middleware. Routes matching exempt patterns are skipped.

WHY IT EXISTS:
  Missing auth middleware on endpoints is a common security vulnerability.
  This rule ensures every new route is either explicitly authenticated
  or explicitly exempted.

CONFIGURATION (.goarch.yml):
  rules:
    authguard:
      router_package: "cmd/api"
      auth_middleware: "middleware.Auth"
      exempt_patterns: ["/health", "/auth/*", "/webhooks/*"]

FIX:
  1. Add the route inside a protected Route group:
     r.Route("/api", func(r chi.Router) {
         r.Use(middleware.Auth)
         r.Get("/users", listUsers)
     })

  2. If the route is intentionally public, add it to exempt_patterns.`,
	},
}

// ruleOrder defines the display order for rules.
var ruleOrder = []string{
	"layerguard", "execguard", "secretguard", "fanout", "methodcount", "apileak",
	"funlen", "argcount", "complexity", "depban", "tagguard", "errguard", "authguard",
}

// Get returns a rule by ID, or nil if not found.
func Get(id string) *Rule {
	r, ok := rules[id]
	if !ok {
		return nil
	}
	return &r
}

// All returns all rules in display order.
func All() []Rule {
	result := make([]Rule, 0, len(ruleOrder))
	for _, id := range ruleOrder {
		result = append(result, rules[id])
	}
	return result
}

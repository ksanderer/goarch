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
    - JSON: json.Marshal(cfg) → secrets in HTTP responses
    - Error messages: fmt.Errorf("failed with cfg: %v", cfg)

  A Secret wrapper type controls how the value is displayed:
    fmt.Println(secret)    → "[REDACTED]"
    json.Marshal(secret)   → "[REDACTED]"
    secret.Value()         → actual value (explicit opt-in)

CONFIGURATION (.goarch.yml):
  rules:
    secretguard:
      type: "Secret"
      field_patterns: ["apikey", "secret", "password", "accesstoken"]
      except_packages: ["internal/handler"]  # response DTOs are OK

FIX:
  1. Create a Secret type in your project:

     type Secret string
     func (s Secret) String() string              { return "[REDACTED]" }
     func (s Secret) MarshalJSON() ([]byte, error) { return []byte(` + "`" + `"[REDACTED]"` + "`" + `), nil }
     func (s Secret) Value() string               { return string(s) }

  2. Change field types:
     // Before
     OpenRouterAPIKey string
     // After
     OpenRouterAPIKey Secret

  3. Use .Value() where the real value is needed:
     req.Header.Set("Authorization", "Bearer " + cfg.APIKey.Value())`,
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
}

// Get returns a rule by ID, or nil if not found.
func Get(id string) *Rule {
	r, ok := rules[id]
	if !ok {
		return nil
	}
	return &r
}

// All returns all rules sorted by ID.
func All() []Rule {
	ids := []string{"layerguard", "execguard", "secretguard", "fanout", "methodcount", "apileak"}
	result := make([]Rule, 0, len(ids))
	for _, id := range ids {
		result = append(result, rules[id])
	}
	return result
}

package config

// Config is the top-level configuration for goarch rules.
type Config struct {
	Rules Rules `yaml:"rules"`
}

// Rules contains all configurable rule sets.
type Rules struct {
	LayerGuard  *LayerGuardConfig  `yaml:"layerguard"`
	ExecGuard   *ExecGuardConfig   `yaml:"execguard"`
	SecretGuard *SecretGuardConfig `yaml:"secretguard"`
	FanOut      *FanOutConfig      `yaml:"fanout"`
	MethodCount *MethodCountConfig `yaml:"methodcount"`
	APILeak     *APILeakConfig     `yaml:"apileak"`
	FunLen      *FunLenConfig      `yaml:"funlen"`
	ArgCount    *ArgCountConfig    `yaml:"argcount"`
	Complexity  *ComplexityConfig  `yaml:"complexity"`
	DepBan      *DepBanConfig      `yaml:"depban"`
	TagGuard    *TagGuardConfig    `yaml:"tagguard"`
	ErrGuard    *ErrGuardConfig    `yaml:"errguard"`
	AuthGuard   *AuthGuardConfig   `yaml:"authguard"`
	External    []ExternalTool     `yaml:"external"`
}

// LayerGuardConfig enforces dependency allowlists/denylists per package.
type LayerGuardConfig struct {
	Layers map[string]LayerRule `yaml:"layers"`
}

type LayerRule struct {
	Allow         []string `yaml:"allow"`
	Deny          []string `yaml:"deny"`
	DenyAllOthers bool     `yaml:"deny_all_others"`
}

// ExecGuardConfig bans specific packages or methods outside allowed packages.
type ExecGuardConfig struct {
	Banned []BannedImport `yaml:"banned"`
}

type BannedImport struct {
	Pkg     string   `yaml:"pkg"`
	Methods []string `yaml:"methods"` // empty = ban entire package
	Except  []string `yaml:"except"`
	Reason  string   `yaml:"reason"`
}

// SecretGuardConfig ensures sensitive fields use a wrapper type.
type SecretGuardConfig struct {
	Type           string   `yaml:"type"`
	FieldPatterns  []string `yaml:"field_patterns"`
	ExceptPackages []string `yaml:"except_packages"` // packages where the rule is not enforced
}

// FanOutConfig limits the number of imports per file.
type FanOutConfig struct {
	MaxImports int `yaml:"max_imports"`
}

// MethodCountConfig limits public methods per type.
type MethodCountConfig struct {
	MaxPublicMethods int `yaml:"max_public_methods"`
}

// APILeakConfig prevents internal types from appearing in public API signatures.
type APILeakConfig struct {
	PublicPackages      []string `yaml:"public_packages"`
	BannedTypesInPublic []string `yaml:"banned_types_in_public"`
}

// FunLenConfig limits function body length.
type FunLenConfig struct {
	MaxLines int `yaml:"max_lines"`
}

// ArgCountConfig limits the number of function parameters.
type ArgCountConfig struct {
	MaxArgs int `yaml:"max_args"`
}

// ComplexityConfig limits cyclomatic complexity per function.
type ComplexityConfig struct {
	MaxComplexity int `yaml:"max_complexity"`
}

// DepBanConfig bans modules in go.mod and optionally limits total dependency count.
type DepBanConfig struct {
	Deny            []ModuleBan `yaml:"deny"`
	MaxDependencies int         `yaml:"max_dependencies"`
}

// ModuleBan describes a single banned module.
type ModuleBan struct {
	Module string `yaml:"module"`
	Reason string `yaml:"reason"`
}

// TagGuardConfig validates struct tag naming conventions per package.
type TagGuardConfig struct {
	Packages map[string]TagRule `yaml:"packages"`
}

// TagRule defines JSON tag requirements for a package.
type TagRule struct {
	JSONNaming      string `yaml:"json_naming"`       // "snake_case" or "camelCase"
	RequireJSONTags bool   `yaml:"require_json_tags"` // exported fields must have json tag
}

// ErrGuardConfig restricts where custom error types can be defined.
type ErrGuardConfig struct {
	AllowedPackages []string `yaml:"allowed_packages"`
}

// AuthGuardConfig verifies endpoint auth middleware coverage.
type AuthGuardConfig struct {
	RouterPackage  string   `yaml:"router_package"`
	AuthMiddleware string   `yaml:"auth_middleware"`
	ExemptPatterns []string `yaml:"exempt_patterns"`
}

// ExternalTool defines a third-party tool to run as part of goarch validation.
type ExternalTool struct {
	Name string `yaml:"name"`
	Cmd  string `yaml:"cmd"`
}

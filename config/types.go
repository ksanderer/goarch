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

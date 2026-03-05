package example

type Secret string

type Config struct {
	Name           string // OK — not sensitive
	APIKey         string // want `\[secretguard\] sensitive field "APIKey" should use type Secret, got string`
	Password       string // want `\[secretguard\] sensitive field "Password" should use type Secret, got string`
	SafeAPIKey     Secret // OK — uses Secret type
	SafePassword   Secret // OK — uses Secret type
	PromptTokens   int    // OK — "token" boundary doesn't match "apikey" or "password"
}

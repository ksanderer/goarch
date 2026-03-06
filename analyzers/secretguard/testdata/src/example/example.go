package example

type Secret string
type SecretBytes []byte

type Config struct {
	Name           string // OK — not sensitive
	APIKey         string // want `\[secretguard\] sensitive field "APIKey" should use one of \[Secret SecretBytes\], got string`
	Password       string // want `\[secretguard\] sensitive field "Password" should use one of \[Secret SecretBytes\], got string`
	SafeAPIKey     Secret // OK — uses Secret type
	SafePassword   Secret // OK — uses Secret type
	PromptTokens   int    // OK — "token" boundary doesn't match "apikey" or "password"
}

type WithPointer struct {
	APIKey *Secret // OK — pointer to Secret type
}

type WithBytes struct {
	APIKey      SecretBytes // OK — uses SecretBytes type
	Password    []byte      // want `\[secretguard\] sensitive field "Password" should use one of \[Secret SecretBytes\], got \[\]byte`
	SafeAPIKey  *SecretBytes // OK — pointer to SecretBytes
}

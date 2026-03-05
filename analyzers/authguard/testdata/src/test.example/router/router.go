package router

import "net/http"

type Router struct{}

func (r *Router) Use(middleware ...func(http.Handler) http.Handler) {}
func (r *Router) Get(pattern string, handler http.HandlerFunc)     {}
func (r *Router) Post(pattern string, handler http.HandlerFunc)    {}
func (r *Router) Route(pattern string, fn func(*Router))           {}

func handler(w http.ResponseWriter, r *http.Request) {}

func Setup() {
	r := &Router{}

	// Exempt route — OK
	r.Get("/health", handler)

	// Exempt by wildcard — OK
	r.Post("/webhooks/stripe", handler)

	// Unprotected route — violation
	r.Get("/users", handler) // want `\[authguard\] route Get "/users" may not have auth middleware`

	// Protected group — Route with Use(Auth) inside
	r.Route("/api", func(sub *Router) {
		sub.Use(Auth)
		sub.Get("/me", handler) // OK — inside group with Auth
	})

	// Unprotected group — Route without Use(Auth)
	r.Route("/admin", func(sub *Router) {
		sub.Get("/dashboard", handler) // want `\[authguard\] route Get "/admin/dashboard" has no auth middleware in its Route group`
	})
}

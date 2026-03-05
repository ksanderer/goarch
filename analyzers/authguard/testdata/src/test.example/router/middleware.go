package router

import "net/http"

func Auth(next http.Handler) http.Handler { return next }

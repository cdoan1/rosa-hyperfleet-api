package middleware

import "net/http"

// AddDeprecationHeaders adds HTTP headers to indicate API deprecation
func AddDeprecationHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-API-Deprecated", "true")
		w.Header().Set("X-API-Deprecated-Version", "v0")
		w.Header().Set("X-API-Current-Version", "v2")
		w.Header().Set("Sunset", "2026-09-23T00:00:00Z")
		w.Header().Set("Link", `</api/v2>; rel="successor-version"`)
		w.Header().Set("Deprecation", "true")
		next.ServeHTTP(w, r)
	})
}

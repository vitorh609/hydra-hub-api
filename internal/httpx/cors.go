package httpx

import (
	"net/http"
	"os"
	"strings"
)

func corsMiddleware() func(http.Handler) http.Handler {
	origins := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if isOriginAllowed(origin, origins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseAllowedOrigins(raw string) map[string]struct{} {
	if strings.TrimSpace(raw) == "" {
		raw = "http://localhost:4200,http://127.0.0.1:4200"
	}

	allowed := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		allowed[origin] = struct{}{}
	}

	return allowed
}

func isOriginAllowed(origin string, allowed map[string]struct{}) bool {
	if _, ok := allowed["*"]; ok {
		return true
	}

	_, ok := allowed[origin]
	return ok
}

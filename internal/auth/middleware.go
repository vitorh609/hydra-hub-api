package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

func RequireAuth(repo *Repo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "bearer token obrigatório"})
				return
			}

			now := time.Now().UTC()
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			user, expiresAt, err := repo.ValidateAndTouchSession(ctx, HashToken(token), now, now.Add(SessionIdleTimeout))
			if err != nil {
				if errors.Is(err, ErrInvalidToken) {
					writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "sessão inválida ou expirada"})
					return
				}

				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}

			w.Header().Set("X-Session-Expires-At", expiresAt.Format(time.RFC3339))
			next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
		})
	}
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", ErrInvalidToken
	}
	return parts[1], nil
}

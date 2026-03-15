package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const SessionIdleTimeout = 15 * time.Minute

type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "json inválido"})
		return
	}

	in.Login = strings.TrimSpace(in.Login)
	if in.Login == "" || in.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "login e password são obrigatórios"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.repo.FindUserByLogin(ctx, in.Login)
	if err != nil || !CheckPassword(in.Password, user.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "login ou senha inválidos"})
		return
	}

	token, tokenHash, err := NewToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "erro ao gerar token"})
		return
	}

	expiresAt := time.Now().UTC().Add(SessionIdleTimeout)
	if err := h.repo.CreateSession(ctx, user.ID, tokenHash, expiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     token,
		Type:      "Bearer",
		ExpiresAt: expiresAt,
		User: SessionUser{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

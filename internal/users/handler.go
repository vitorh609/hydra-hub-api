package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"api-hydra-hub/internal/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "json inválido"})
		return
	}
	passwordHash := passwordHashFromInput(in.Password, in.PasswordHash)
	if in.Name == "" || in.Email == "" || passwordHash == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name, email e password são obrigatórios"})
		return
	}
	in.PasswordHash = passwordHash
	in.Password = ""

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	u, err := h.repo.Create(ctx, in)
	if err != nil {
		// pode estourar unique violation no email
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, u)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	users, err := h.repo.List(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	u, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "user não encontrado"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "json inválido"})
		return
	}

	if in.Password != nil || in.PasswordHash != nil {
		passwordHash := passwordHashFromInput(optionalStringValue(in.Password), optionalStringValue(in.PasswordHash))
		if passwordHash != "" {
			in.PasswordHash = &passwordHash
		} else {
			in.PasswordHash = nil
		}
		in.Password = nil
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	u, err := h.repo.Update(ctx, id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "user não encontrado"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "user não encontrado"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func passwordHashFromInput(password string, passwordHash string) string {
	password = strings.TrimSpace(password)
	passwordHash = strings.TrimSpace(passwordHash)

	if password != "" {
		return auth.HashPassword(password)
	}

	return passwordHash
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

package clickup

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"api-hydra-hub/internal/auth"

	"github.com/go-chi/chi/v5"
)

type ServiceContract interface {
	Connect(ctx context.Context, userID string, in ConnectClickupDto) (ClickupConnectionStatusDto, error)
	Status(ctx context.Context, userID string) (ClickupConnectionStatusDto, error)
	ListSpaces(ctx context.Context, userID string) ([]ClickupSpaceDto, error)
	ListFolders(ctx context.Context, userID string, spaceID string) (ClickupFoldersResponseDto, error)
	ListTasks(ctx context.Context, userID string, listID string, page int) (ClickupTasksResponseDto, error)
	CreateTask(ctx context.Context, userID string, listID string, in CreateClickupTaskDto) (ClickupTaskDto, error)
}

type Handler struct {
	service ServiceContract
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "usuário não autenticado"})
		return
	}

	var in ConnectClickupDto
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "json inválido"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	status, err := h.service.Connect(ctx, user.ID, in)
	if err != nil {
		writeIntegrationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "usuário não autenticado"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	status, err := h.service.Status(ctx, user.ID)
	if err != nil {
		writeIntegrationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) ListSpaces(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "usuário não autenticado"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	spaces, err := h.service.ListSpaces(ctx, user.ID)
	if err != nil {
		writeIntegrationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, spaces)
}

func (h *Handler) ListFolders(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "usuário não autenticado"})
		return
	}

	spaceID := strings.TrimSpace(chi.URLParam(r, "spaceId"))
	if spaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "spaceId é obrigatório"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	response, err := h.service.ListFolders(ctx, user.ID, spaceID)
	if err != nil {
		writeIntegrationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "usuário não autenticado"})
		return
	}

	listID := strings.TrimSpace(chi.URLParam(r, "listId"))
	if listID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "listId é obrigatório"})
		return
	}

	page := 0
	if rawPage := strings.TrimSpace(r.URL.Query().Get("page")); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "page deve ser um inteiro maior ou igual a zero"})
			return
		}
		page = parsed
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	response, err := h.service.ListTasks(ctx, user.ID, listID, page)
	if err != nil {
		writeIntegrationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "usuário não autenticado"})
		return
	}

	listID := strings.TrimSpace(chi.URLParam(r, "listId"))
	if listID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "listId é obrigatório"})
		return
	}

	var in CreateClickupTaskDto
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "json inválido"})
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name é obrigatório"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	task, err := h.service.CreateTask(ctx, user.ID, listID, in)
	if err != nil {
		writeIntegrationError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func writeIntegrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrIntegrationDisabled):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "integração ClickUp indisponível"})
	case errors.Is(err, ErrConnectionNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "integração ClickUp não configurada"})
	case errors.Is(err, ErrInvalidCredentials):
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "credenciais do ClickUp inválidas"})
	case errors.Is(err, ErrUpstreamUnavailable):
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "ClickUp temporariamente indisponível"})
	case errors.Is(err, ErrBadGateway):
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "falha ao comunicar com o ClickUp"})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

package httpx

import (
	"net/http"

	"api-hydra-hub/internal/account-settings"
	"api-hydra-hub/internal/notes"
	"api-hydra-hub/internal/users"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware())

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	userRepo := users.NewRepo(pool)
	userHandler := users.NewHandler(userRepo)

	noteRepo := notes.NewRepo(pool)
	noteHandler := notes.NewHandler(noteRepo)

	accountSettingsRepo := account_settings.NewRepo(pool)
	accountSettingsHandler := account_settings.NewHandler(accountSettingsRepo)

	r.Route("/users", func(r chi.Router) {
		r.Post("/", userHandler.Create)
		r.Get("/", userHandler.List)
		r.Get("/{id}", userHandler.GetByID)
		r.Put("/{id}", userHandler.Update)
		r.Delete("/{id}", userHandler.Delete)
	})

	r.Route("/notes", func(r chi.Router) {
		r.Post("/", noteHandler.Create)
		r.Get("/", noteHandler.List)
		r.Get("/{id}", noteHandler.GetByID)
		r.Put("/{id}", noteHandler.Update)
		r.Delete("/{id}", noteHandler.Delete)
	})

	r.Route("/account-settings", func(r chi.Router) {
		r.Post("/", accountSettingsHandler.Create)
		r.Get("/", accountSettingsHandler.List)
		r.Get("/{id}", accountSettingsHandler.GetByID)
		r.Put("/{id}", accountSettingsHandler.Update)
		r.Delete("/{id}", accountSettingsHandler.Delete)
	})

	return r
}

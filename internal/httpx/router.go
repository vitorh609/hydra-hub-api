package httpx

import (
	"context"
	"net/http"
	"os"
	"time"

	"api-hydra-hub/internal/account-settings"
	"api-hydra-hub/internal/auth"
	"api-hydra-hub/internal/integrations/clickup"
	"api-hydra-hub/internal/notes"
	"api-hydra-hub/internal/tickets"
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
	authRepo := auth.NewRepo(pool)
	authHandler := auth.NewHandler(authRepo)

	noteRepo := notes.NewRepo(pool)
	noteHandler := notes.NewHandler(noteRepo)

	ticketRepo := tickets.NewRepo(pool)
	ticketHandler := tickets.NewHandler(ticketRepo)

	accountSettingsRepo := account_settings.NewRepo(pool)
	accountSettingsHandler := account_settings.NewHandler(accountSettingsRepo)

	clickupRepo := clickup.NewRepo(pool)
	schemaCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := clickupRepo.EnsureSchema(schemaCtx); err != nil {
		panic(err)
	}
	clickupCipher, err := clickup.NewCredentialCipher(os.Getenv("CLICKUP_CREDENTIALS_ENCRYPTION_KEY"), "v1")
	if err != nil {
		panic(err)
	}
	clickupClient := clickup.NewClient(nil)
	clickupService := clickup.NewService(clickupRepo, clickupClient, clickupCipher, nil)
	clickupHandler := clickup.NewHandler(clickupService)

	r.Post("/auth/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(authRepo))

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

		r.Route("/tickets", func(r chi.Router) {
			r.Post("/", ticketHandler.Create)
			r.Get("/", ticketHandler.List)
			r.Get("/{id}", ticketHandler.GetByID)
			r.Put("/{id}", ticketHandler.Update)
			r.Delete("/{id}", ticketHandler.Delete)
		})

		r.Route("/account-settings", func(r chi.Router) {
			r.Post("/", accountSettingsHandler.Create)
			r.Get("/", accountSettingsHandler.List)
			r.Get("/{id}", accountSettingsHandler.GetByID)
			r.Put("/{id}", accountSettingsHandler.Update)
			r.Delete("/{id}", accountSettingsHandler.Delete)
		})

		r.Route("/integrations/clickup", func(r chi.Router) {
			r.Post("/connect", clickupHandler.Connect)
			r.Get("/status", clickupHandler.Status)
			r.Get("/spaces", clickupHandler.ListSpaces)
			r.Get("/spaces/{spaceId}/folders", clickupHandler.ListFolders)
			r.Get("/lists/{listId}/tasks", clickupHandler.ListTasks)
			r.Post("/lists/{listId}/tasks", clickupHandler.CreateTask)
		})
	})

	return r
}

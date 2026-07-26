package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/open-services-lab/customer-registry-api/internal/handlers"
	appmiddleware "github.com/open-services-lab/customer-registry-api/internal/middleware"
	"github.com/open-services-lab/customer-registry-api/internal/models"
	"github.com/open-services-lab/customer-registry-api/internal/storage"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("configurazione non valida", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := storage.Open(ctx, config.DBPath)
	if err != nil {
		slog.Error("impossibile inizializzare il database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.EnsureBootstrapUsers(
		ctx,
		config.AdminEmail,
		config.AdminPassword,
		config.OperatorEmail,
		config.OperatorPassword,
	); err != nil {
		slog.Error("impossibile creare gli utenti iniziali", "error", err)
		os.Exit(1)
	}
	if err := store.DeleteExpiredTokens(ctx); err != nil {
		slog.Warn("pulizia token scaduti non riuscita", "error", err)
	}

	if config.UsingDefaultCredentials {
		slog.Warn("sono attive le credenziali demo; impostare APP_ADMIN_* e APP_OPERATOR_* fuori dallo sviluppo locale")
	}

	router := buildRouter(store, config.TokenTTL)
	server := &http.Server{
		Addr:              config.Address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("API avviata", "address", config.Address, "database", config.DBPath)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case signal := <-shutdownSignals:
		slog.Info("arresto richiesto", "signal", signal.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server terminato in modo inatteso", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("arresto forzato del server", "error", err)
		_ = server.Close()
	}
}

func buildRouter(store *storage.Store, tokenTTL time.Duration) http.Handler {
	authHandler := handlers.NewAuthHandler(store, tokenTTL)
	clientHandler := handlers.NewClientHandler(store)
	contactHandler := handlers.NewContactHandler(store)
	authenticator := appmiddleware.NewAuthenticator(store)

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Logger)
	router.Use(securityHeaders)
	router.Use(recoverJSON)

	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeRouterError(w, http.StatusNotFound, "route_not_found", "percorso API non trovato")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeRouterError(w, http.StatusMethodNotAllowed, "method_not_allowed", "metodo HTTP non consentito per questo percorso")
	})

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := store.Ping(r.Context()); err != nil {
			writeRouterError(w, http.StatusServiceUnavailable, "database_unavailable", "database non disponibile")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/login", authHandler.Login)

		api.Group(func(protected chi.Router) {
			protected.Use(authenticator.Authenticate)
			protected.Post("/auth/logout", authHandler.Logout)

			protected.Route("/clienti", func(clients chi.Router) {
				clients.Get("/", clientHandler.List)
				clients.Post("/", clientHandler.Create)
				clients.Get("/{clientID}", clientHandler.Get)
				clients.With(appmiddleware.RequireRole(models.RoleAdministrator)).Put("/{clientID}", clientHandler.Update)
				clients.With(appmiddleware.RequireRole(models.RoleAdministrator)).Delete("/{clientID}", clientHandler.Delete)

				clients.Get("/{clientID}/referenti", contactHandler.List)
				clients.Post("/{clientID}/referenti", contactHandler.Create)
				clients.Get("/{clientID}/referenti/{contactID}", contactHandler.Get)
				clients.With(appmiddleware.RequireRole(models.RoleAdministrator)).Put("/{clientID}/referenti/{contactID}", contactHandler.Update)
				clients.With(appmiddleware.RequireRole(models.RoleAdministrator)).Delete("/{clientID}/referenti/{contactID}", contactHandler.Delete)
			})
		})
	})

	return router
}

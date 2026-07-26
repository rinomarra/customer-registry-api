package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/rinomarra/customer-registry-api/internal/models"
	"github.com/rinomarra/customer-registry-api/internal/storage"
)

type contextKey string

const principalKey contextKey = "authenticated-principal"

type Principal struct {
	User  models.User
	Token string
}

type Authenticator struct {
	store *storage.Store
}

func NewAuthenticator(store *storage.Store) *Authenticator {
	return &Authenticator{store: store}
}

func (a *Authenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "unauthorized", "token di autenticazione mancante o non valido")
			return
		}

		user, err := a.store.UserByToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, storage.ErrInvalidCredentials) {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "token scaduto o non valido")
				return
			}
			writeAuthError(w, http.StatusInternalServerError, "internal_error", "errore interno durante l'autenticazione")
			return
		}

		principal := Principal{User: user, Token: token}
		ctx := context.WithValue(r.Context(), principalKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(roles ...models.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[models.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "autenticazione richiesta")
				return
			}
			if _, ok := allowed[principal.User.Role]; !ok {
				writeAuthError(w, http.StatusForbidden, "forbidden", "il ruolo corrente non può eseguire questa operazione")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey).(Principal)
	return principal, ok
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

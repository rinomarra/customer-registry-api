package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	appmiddleware "github.com/open-services-lab/customer-registry-api/internal/middleware"
	"github.com/open-services-lab/customer-registry-api/internal/models"
	"github.com/open-services-lab/customer-registry-api/internal/storage"
)

type AuthHandler struct {
	store    *storage.Store
	tokenTTL time.Duration
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"scade_il"`
	User      models.User `json:"utente"`
}

func NewAuthHandler(store *storage.Store, tokenTTL time.Duration) *AuthHandler {
	return &AuthHandler{store: store, tokenTTL: tokenTTL}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	validation := models.ValidationErrors{}
	if request.Email == "" {
		validation["email"] = "è obbligatoria"
	} else if !models.IsValidEmail(request.Email) {
		validation["email"] = "deve essere un indirizzo email valido"
	}
	if request.Password == "" {
		validation["password"] = "è obbligatoria"
	}
	if validation.HasErrors() {
		writeError(w, http.StatusBadRequest, "validation_failed", "i dati di accesso non sono validi", validation)
		return
	}

	user, err := h.store.AuthenticateUser(r.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "email o password non corretti", nil)
			return
		}
		writeInternalError(w, r, err)
		return
	}

	token, err := h.store.CreateToken(r.Context(), user.ID, h.tokenTTL)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, dataEnvelope{Data: loginResponse{
		Token:     token.Value,
		ExpiresAt: token.ExpiresAt,
		User:      user,
	}})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	principal, ok := appmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "autenticazione richiesta", nil)
		return
	}
	if err := h.store.RevokeToken(r.Context(), principal.Token); err != nil {
		writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/open-services-lab/customer-registry-api/internal/models"
	"github.com/open-services-lab/customer-registry-api/internal/storage"
)

type ClientHandler struct {
	store *storage.Store
}

func NewClientHandler(store *storage.Store) *ClientHandler {
	return &ClientHandler{store: store}
}

func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("stato")))
	if status != "" && !models.ClientStatus(status).IsValid() {
		writeError(w, http.StatusBadRequest, "validation_failed", "il filtro stato non è valido", map[string]string{
			"stato": "deve essere attivo oppure sospeso",
		})
		return
	}

	vatNumber := strings.TrimSpace(r.URL.Query().Get("partita_iva"))
	if vatNumber != "" && !models.IsValidVATNumber(vatNumber) {
		writeError(w, http.StatusBadRequest, "validation_failed", "il filtro partita IVA non è valido", map[string]string{
			"partita_iva": "deve contenere esattamente 11 cifre",
		})
		return
	}

	clients, total, err := h.store.ListClients(r.Context(), storage.ClientFilter{
		Page:      page,
		PageSize:  pageSize,
		Query:     strings.TrimSpace(r.URL.Query().Get("q")),
		VATNumber: vatNumber,
		Status:    status,
	})
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, listEnvelope{
		Data:       clients,
		Pagination: newPagination(page, pageSize, total),
	})
}

func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	clientID, err := parsePathID(r, "clientID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	client, err := h.store.GetClient(r.Context(), clientID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "client_not_found", "cliente non trovato", nil)
			return
		}
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: client})
}

func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input models.ClientInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}

	normalized, validation := validateClientInput(input, true)
	if validation.HasErrors() {
		writeError(w, http.StatusBadRequest, "validation_failed", "i dati del cliente non sono validi", validation)
		return
	}

	client, err := h.store.CreateClient(r.Context(), normalized)
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			writeError(w, http.StatusConflict, "vat_number_conflict", "esiste già un cliente con questa partita IVA", map[string]string{
				"partita_iva": "deve essere univoca",
			})
			return
		}
		writeInternalError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/clienti/"+formatID(client.ID))
	writeJSON(w, http.StatusCreated, dataEnvelope{Data: client})
}

func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	clientID, err := parsePathID(r, "clientID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	var input models.ClientInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}

	normalized, validation := validateClientInput(input, false)
	if validation.HasErrors() {
		writeError(w, http.StatusBadRequest, "validation_failed", "i dati del cliente non sono validi", validation)
		return
	}

	client, err := h.store.UpdateClient(r.Context(), clientID, normalized)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNotFound):
			writeError(w, http.StatusNotFound, "client_not_found", "cliente non trovato", nil)
		case errors.Is(err, storage.ErrConflict):
			writeError(w, http.StatusConflict, "vat_number_conflict", "esiste già un cliente con questa partita IVA", map[string]string{
				"partita_iva": "deve essere univoca",
			})
		default:
			writeInternalError(w, r, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, dataEnvelope{Data: client})
}

func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	clientID, err := parsePathID(r, "clientID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	if err := h.store.DeleteClient(r.Context(), clientID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "client_not_found", "cliente non trovato", nil)
			return
		}
		writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validateClientInput rende esplicita la validazione applicata dagli handler.
// I vincoli sono duplicati nei modelli di dominio per poter essere riutilizzati
// anche da job, importatori o interfacce diverse dall'HTTP.
func validateClientInput(input models.ClientInput, creating bool) (models.ClientInput, models.ValidationErrors) {
	input = input.Normalize()
	errs := models.ValidationErrors{}

	if input.BusinessName == "" {
		errs["ragione_sociale"] = "è obbligatoria"
	} else if len([]rune(input.BusinessName)) > models.MaxBusinessNameLength {
		errs["ragione_sociale"] = "non può superare 200 caratteri"
	}

	if input.VATNumber == "" {
		errs["partita_iva"] = "è obbligatoria"
	} else if !models.IsValidVATNumber(input.VATNumber) {
		errs["partita_iva"] = "deve contenere esattamente 11 cifre"
	}

	if input.Email == "" {
		errs["email"] = "è obbligatoria"
	} else if len(input.Email) > models.MaxEmailLength {
		errs["email"] = "non può superare 254 caratteri"
	} else if !models.IsValidEmail(input.Email) {
		errs["email"] = "deve essere un indirizzo email valido"
	}

	if input.Phone != "" {
		if len([]rune(input.Phone)) > models.MaxPhoneLength {
			errs["telefono"] = "non può superare 30 caratteri"
		} else {
			for _, character := range input.Phone {
				if !strings.ContainsRune("0123456789+(). /-", character) {
					errs["telefono"] = "può contenere solo cifre, spazi e i simboli + ( ) . / -"
					break
				}
			}
		}
	}

	if len([]rune(input.Address)) > models.MaxAddressLength {
		errs["indirizzo"] = "non può superare 500 caratteri"
	}

	if len([]rune(input.Notes)) > models.MaxNotesLength {
		errs["note"] = "non possono superare 2000 caratteri"
	}

	if creating && input.Status == "" {
		input.Status = models.ClientStatusActive
	}
	if input.Status == "" {
		errs["stato"] = "è obbligatorio in modifica"
	} else if !input.Status.IsValid() {
		errs["stato"] = "deve essere attivo oppure sospeso"
	}

	return input, errs
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

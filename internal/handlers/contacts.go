package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/open-services-lab/customer-registry-api/internal/models"
	"github.com/open-services-lab/customer-registry-api/internal/storage"
)

type ContactHandler struct {
	store *storage.Store
}

func NewContactHandler(store *storage.Store) *ContactHandler {
	return &ContactHandler{store: store}
}

func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	clientID, err := parsePathID(r, "clientID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}
	if _, err := h.store.GetClient(r.Context(), clientID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "client_not_found", "cliente non trovato", nil)
			return
		}
		writeInternalError(w, r, err)
		return
	}

	page, pageSize, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	contacts, total, err := h.store.ListContacts(r.Context(), clientID, storage.ContactFilter{
		Page:     page,
		PageSize: pageSize,
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
	})
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, listEnvelope{
		Data:       contacts,
		Pagination: newPagination(page, pageSize, total),
	})
}

func (h *ContactHandler) Get(w http.ResponseWriter, r *http.Request) {
	clientID, contactID, ok := parseContactIDs(w, r)
	if !ok {
		return
	}

	contact, err := h.store.GetContact(r.Context(), clientID, contactID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contact_not_found", "referente non trovato per il cliente indicato", nil)
			return
		}
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: contact})
}

func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	clientID, err := parsePathID(r, "clientID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	var input models.ContactInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	input = input.Normalize()
	if validation := input.Validate(); validation.HasErrors() {
		writeError(w, http.StatusBadRequest, "validation_failed", "i dati del referente non sono validi", validation)
		return
	}

	contact, err := h.store.CreateContact(r.Context(), clientID, input)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "client_not_found", "cliente non trovato", nil)
			return
		}
		writeInternalError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/clienti/"+formatID(clientID)+"/referenti/"+formatID(contact.ID))
	writeJSON(w, http.StatusCreated, dataEnvelope{Data: contact})
}

func (h *ContactHandler) Update(w http.ResponseWriter, r *http.Request) {
	clientID, contactID, ok := parseContactIDs(w, r)
	if !ok {
		return
	}

	var input models.ContactInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	input = input.Normalize()
	if validation := input.Validate(); validation.HasErrors() {
		writeError(w, http.StatusBadRequest, "validation_failed", "i dati del referente non sono validi", validation)
		return
	}

	contact, err := h.store.UpdateContact(r.Context(), clientID, contactID, input)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contact_not_found", "referente non trovato per il cliente indicato", nil)
			return
		}
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dataEnvelope{Data: contact})
}

func (h *ContactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	clientID, contactID, ok := parseContactIDs(w, r)
	if !ok {
		return
	}

	if err := h.store.DeleteContact(r.Context(), clientID, contactID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "contact_not_found", "referente non trovato per il cliente indicato", nil)
			return
		}
		writeInternalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseContactIDs(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	clientID, err := parsePathID(r, "clientID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return 0, 0, false
	}
	contactID, err := parsePathID(r, "contactID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return 0, 0, false
	}
	return clientID, contactID, true
}

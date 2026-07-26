package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

const maxRequestBodyBytes = 1 << 20

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type dataEnvelope struct {
	Data any `json:"data"`
}

type listEnvelope struct {
	Data       any        `json:"data"`
	Pagination pagination `json:"paginazione"`
}

type pagination struct {
	Page       int `json:"pagina"`
	PageSize   int `json:"elementi_per_pagina"`
	Total      int `json:"totale"`
	TotalPages int `json:"pagine_totali"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		var syntaxError *json.SyntaxError
		var typeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("JSON non valido alla posizione %d", syntaxError.Offset)
		case errors.As(err, &typeError):
			return fmt.Errorf("valore non valido per il campo %q", typeError.Field)
		case errors.Is(err, io.EOF):
			return errors.New("il corpo della richiesta è obbligatorio")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			return fmt.Errorf("campo JSON non riconosciuto: %s", err.Error()[20:])
		case errors.As(err, &maxBytesError):
			return errors.New("il corpo della richiesta supera 1 MiB")
		default:
			return errors.New("corpo JSON non valido")
		}
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("il corpo deve contenere un solo oggetto JSON")
	}
	return nil
}

func parsePathID(r *http.Request, name string) (int64, error) {
	value := chi.URLParam(r, name)
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("identificativo %s non valido", name)
	}
	return id, nil
}

func parsePagination(r *http.Request) (int, int, error) {
	page, err := parsePositiveIntQuery(r, "pagina", 1)
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := parsePositiveIntQuery(r, "elementi_per_pagina", 20)
	if err != nil {
		return 0, 0, err
	}
	if pageSize > 100 {
		return 0, 0, errors.New("elementi_per_pagina non può superare 100")
	}
	return page, pageSize, nil
}

func parsePositiveIntQuery(r *http.Request, name string, defaultValue int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s deve essere un intero positivo", name)
	}
	return value, nil
}

func newPagination(page, pageSize, total int) pagination {
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string, details map[string]string) {
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	writeJSON(w, status, errorEnvelope{Error: apiError{
		Code:    code,
		Message: message,
		Details: details,
	}})
}

func writeInternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("errore API", "method", r.Method, "path", r.URL.Path, "error", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "si è verificato un errore interno", nil)
}

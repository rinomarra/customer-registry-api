package storage

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("risorsa non trovata")
	ErrConflict           = errors.New("conflitto sui dati")
	ErrInvalidCredentials = errors.New("credenziali non valide")
)

type ClientFilter struct {
	Page      int
	PageSize  int
	Query     string
	VATNumber string
	Status    string
}

type ContactFilter struct {
	Page     int
	PageSize int
	Query    string
}

type Token struct {
	Value     string
	ExpiresAt time.Time
}

package models

import (
	"net/mail"
	"regexp"
	"strings"
	"time"
)

type ClientStatus string

const (
	ClientStatusActive    ClientStatus = "attivo"
	ClientStatusSuspended ClientStatus = "sospeso"

	MaxBusinessNameLength = 200
	MaxEmailLength        = 254
	MaxPhoneLength        = 30
	MaxAddressLength      = 500
	MaxNotesLength        = 2000
)

var (
	vatNumberPattern = regexp.MustCompile(`^[0-9]{11}$`)
	phonePattern     = regexp.MustCompile(`^[0-9+(). /-]+$`)
)

type Client struct {
	ID           int64        `json:"id"`
	BusinessName string       `json:"ragione_sociale"`
	VATNumber    string       `json:"partita_iva"`
	Email        string       `json:"email"`
	Phone        string       `json:"telefono,omitempty"`
	Address      string       `json:"indirizzo,omitempty"`
	Notes        string       `json:"note,omitempty"`
	Status       ClientStatus `json:"stato"`
	CreatedAt    time.Time    `json:"creato_il"`
	UpdatedAt    time.Time    `json:"aggiornato_il"`
}

type ClientInput struct {
	BusinessName string       `json:"ragione_sociale"`
	VATNumber    string       `json:"partita_iva"`
	Email        string       `json:"email"`
	Phone        string       `json:"telefono"`
	Address      string       `json:"indirizzo"`
	Notes        string       `json:"note"`
	Status       ClientStatus `json:"stato"`
}

func (in ClientInput) Normalize() ClientInput {
	in.BusinessName = strings.TrimSpace(in.BusinessName)
	in.VATNumber = strings.TrimSpace(in.VATNumber)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.Address = strings.TrimSpace(in.Address)
	in.Notes = strings.TrimSpace(in.Notes)
	in.Status = ClientStatus(strings.ToLower(strings.TrimSpace(string(in.Status))))
	return in
}

func (in ClientInput) ValidateForCreate() ValidationErrors {
	normalized := in.Normalize()
	if normalized.Status == "" {
		normalized.Status = ClientStatusActive
	}
	return validateClientInput(normalized)
}

func (in ClientInput) ValidateForUpdate() ValidationErrors {
	return validateClientInput(in.Normalize())
}

func validateClientInput(in ClientInput) ValidationErrors {
	errs := ValidationErrors{}

	if in.BusinessName == "" {
		errs["ragione_sociale"] = "è obbligatoria"
	} else if len([]rune(in.BusinessName)) > MaxBusinessNameLength {
		errs["ragione_sociale"] = "non può superare 200 caratteri"
	}

	if in.VATNumber == "" {
		errs["partita_iva"] = "è obbligatoria"
	} else if !IsValidVATNumber(in.VATNumber) {
		errs["partita_iva"] = "deve contenere esattamente 11 cifre"
	}

	if in.Email == "" {
		errs["email"] = "è obbligatoria"
	} else if !IsValidEmail(in.Email) {
		errs["email"] = "deve essere un indirizzo email valido"
	}

	if in.Phone != "" {
		if len([]rune(in.Phone)) > MaxPhoneLength {
			errs["telefono"] = "non può superare 30 caratteri"
		} else if !phonePattern.MatchString(in.Phone) {
			errs["telefono"] = "può contenere solo cifre, spazi e i simboli + ( ) . / -"
		}
	}

	if len([]rune(in.Address)) > MaxAddressLength {
		errs["indirizzo"] = "non può superare 500 caratteri"
	}

	if len([]rune(in.Notes)) > MaxNotesLength {
		errs["note"] = "non possono superare 2000 caratteri"
	}

	if !in.Status.IsValid() {
		errs["stato"] = "deve essere attivo oppure sospeso"
	}

	return errs
}

func (s ClientStatus) IsValid() bool {
	return s == ClientStatusActive || s == ClientStatusSuspended
}

func IsValidVATNumber(value string) bool {
	return vatNumberPattern.MatchString(value)
}

func IsValidEmail(value string) bool {
	if value == "" || len(value) > MaxEmailLength || strings.ContainsAny(value, "\r\n") {
		return false
	}
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value
}

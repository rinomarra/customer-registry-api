package models

import (
	"strings"
	"time"
)

const (
	MaxContactNameLength = 100
	MaxContactRoleLength = 100
)

type Contact struct {
	ID        int64     `json:"id"`
	ClientID  int64     `json:"cliente_id"`
	FirstName string    `json:"nome"`
	LastName  string    `json:"cognome"`
	Email     string    `json:"email"`
	Role      string    `json:"ruolo,omitempty"`
	CreatedAt time.Time `json:"creato_il"`
	UpdatedAt time.Time `json:"aggiornato_il"`
}

type ContactInput struct {
	FirstName string `json:"nome"`
	LastName  string `json:"cognome"`
	Email     string `json:"email"`
	Role      string `json:"ruolo"`
}

func (in ContactInput) Normalize() ContactInput {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Role = strings.TrimSpace(in.Role)
	return in
}

func (in ContactInput) Validate() ValidationErrors {
	in = in.Normalize()
	errs := ValidationErrors{}

	if in.FirstName == "" {
		errs["nome"] = "è obbligatorio"
	} else if len([]rune(in.FirstName)) > MaxContactNameLength {
		errs["nome"] = "non può superare 100 caratteri"
	}

	if in.LastName == "" {
		errs["cognome"] = "è obbligatorio"
	} else if len([]rune(in.LastName)) > MaxContactNameLength {
		errs["cognome"] = "non può superare 100 caratteri"
	}

	if in.Email == "" {
		errs["email"] = "è obbligatoria"
	} else if !IsValidEmail(in.Email) {
		errs["email"] = "deve essere un indirizzo email valido"
	}

	if len([]rune(in.Role)) > MaxContactRoleLength {
		errs["ruolo"] = "non può superare 100 caratteri"
	}

	return errs
}

package models

import "time"

type UserRole string

const (
	RoleAdministrator UserRole = "amministratore"
	RoleOperator      UserRole = "operatore"
)

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"ruolo"`
	Active    bool      `json:"attivo"`
	CreatedAt time.Time `json:"creato_il"`
}

func (r UserRole) IsValid() bool {
	return r == RoleAdministrator || r == RoleOperator
}

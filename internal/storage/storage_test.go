package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rinomarra/customer-registry-api/internal/models"
)

func TestClientVATConflictAndCascadeDelete(t *testing.T) {
	store, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("apertura database: %v", err)
	}
	defer store.Close()

	input := models.ClientInput{
		BusinessName: "Acme S.r.l.",
		VATNumber:    "12345678901",
		Email:        "info@acme.test",
		Status:       models.ClientStatusActive,
	}
	client, err := store.CreateClient(context.Background(), input)
	if err != nil {
		t.Fatalf("creazione cliente: %v", err)
	}

	if _, err := store.CreateClient(context.Background(), input); !errors.Is(err, ErrConflict) {
		t.Fatalf("atteso ErrConflict, ottenuto %v", err)
	}

	contact, err := store.CreateContact(context.Background(), client.ID, models.ContactInput{
		FirstName: "Mario",
		LastName:  "Rossi",
		Email:     "mario.rossi@acme.test",
		Role:      "Responsabile acquisti",
	})
	if err != nil {
		t.Fatalf("creazione referente: %v", err)
	}

	if err := store.DeleteClient(context.Background(), client.ID); err != nil {
		t.Fatalf("cancellazione cliente: %v", err)
	}
	if _, err := store.GetContact(context.Background(), client.ID, contact.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("il referente doveva essere cancellato a cascata, ottenuto %v", err)
	}
}

func TestAuthenticationTokenLifecycle(t *testing.T) {
	store, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("apertura database: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.EnsureBootstrapUsers(
		ctx,
		"admin@example.local",
		"Admin123!",
		"operator@example.local",
		"Operator123!",
	); err != nil {
		t.Fatalf("creazione utenti iniziali: %v", err)
	}

	user, err := store.AuthenticateUser(ctx, "ADMIN@example.local", "Admin123!")
	if err != nil {
		t.Fatalf("autenticazione: %v", err)
	}
	if user.Role != models.RoleAdministrator {
		t.Fatalf("ruolo inatteso: %s", user.Role)
	}

	token, err := store.CreateToken(ctx, user.ID, time.Hour)
	if err != nil {
		t.Fatalf("creazione token: %v", err)
	}
	if _, err := store.UserByToken(ctx, token.Value); err != nil {
		t.Fatalf("token valido rifiutato: %v", err)
	}

	if err := store.RevokeToken(ctx, token.Value); err != nil {
		t.Fatalf("revoca token: %v", err)
	}
	if _, err := store.UserByToken(ctx, token.Value); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("token revocato ancora valido: %v", err)
	}
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := hashPassword("Password123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !verifyPassword(hash, "Password123!") {
		t.Fatal("la password corretta non è stata verificata")
	}
	if verifyPassword(hash, "PasswordErrata123!") {
		t.Fatal("una password errata è stata accettata")
	}
}

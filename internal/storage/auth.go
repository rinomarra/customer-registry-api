package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/rinomarra/customer-registry-api/internal/models"
)

func (s *Store) EnsureBootstrapUsers(
	ctx context.Context,
	adminEmail string,
	adminPassword string,
	operatorEmail string,
	operatorPassword string,
) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("conteggio utenti: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("avvio transazione utenti iniziali: %w", err)
	}
	defer tx.Rollback()

	if err := insertUser(ctx, tx, adminEmail, adminPassword, models.RoleAdministrator); err != nil {
		return err
	}
	if err := insertUser(ctx, tx, operatorEmail, operatorPassword, models.RoleOperator); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("salvataggio utenti iniziali: %w", err)
	}
	return nil
}

func insertUser(ctx context.Context, tx *sql.Tx, email, password string, role models.UserRole) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !models.IsValidEmail(email) {
		return fmt.Errorf("email utente iniziale non valida: %s", email)
	}
	if len(password) < 8 {
		return fmt.Errorf("la password iniziale per %s deve avere almeno 8 caratteri", email)
	}
	if !role.IsValid() {
		return fmt.Errorf("ruolo iniziale non valido: %s", role)
	}

	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password iniziale: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, role, active, created_at)
		VALUES (?, ?, ?, 1, ?)
	`, email, hash, role, formatTime(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("creazione utente iniziale %s: %w", email, err)
	}
	return nil
}

func (s *Store) AuthenticateUser(ctx context.Context, email, password string) (models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	var passwordHash string
	var active int
	var role string
	var createdAt string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, active, created_at
		FROM users
		WHERE email = ? COLLATE NOCASE
	`, email).Scan(&user.ID, &user.Email, &passwordHash, &role, &active, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, ErrInvalidCredentials
		}
		return models.User{}, fmt.Errorf("lettura utente: %w", err)
	}

	if active != 1 || !verifyPassword(passwordHash, password) {
		return models.User{}, ErrInvalidCredentials
	}

	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return models.User{}, err
	}
	user.Role = models.UserRole(role)
	user.Active = true
	user.CreatedAt = parsedCreatedAt
	return user, nil
}

func (s *Store) CreateToken(ctx context.Context, userID int64, ttl time.Duration) (Token, error) {
	if ttl <= 0 {
		return Token{}, fmt.Errorf("durata token non valida")
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Token{}, fmt.Errorf("generazione token: %w", err)
	}
	value := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().UTC().Add(ttl)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_tokens (user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, userID, hashToken(value), formatTime(expiresAt), formatTime(time.Now().UTC()))
	if err != nil {
		return Token{}, fmt.Errorf("salvataggio token: %w", err)
	}

	return Token{Value: value, ExpiresAt: expiresAt}, nil
}

func (s *Store) UserByToken(ctx context.Context, rawToken string) (models.User, error) {
	var user models.User
	var role string
	var active int
	var userCreatedAt string
	var expiresAt string

	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.role, u.active, u.created_at, t.expires_at
		FROM auth_tokens t
		JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = ?
	`, hashToken(rawToken)).Scan(
		&user.ID,
		&user.Email,
		&role,
		&active,
		&userCreatedAt,
		&expiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, ErrInvalidCredentials
		}
		return models.User{}, fmt.Errorf("lettura token: %w", err)
	}

	expiry, err := parseTime(expiresAt)
	if err != nil {
		return models.User{}, err
	}
	if active != 1 || !expiry.After(time.Now().UTC()) {
		_ = s.RevokeToken(ctx, rawToken)
		return models.User{}, ErrInvalidCredentials
	}

	createdAt, err := parseTime(userCreatedAt)
	if err != nil {
		return models.User{}, err
	}
	user.Role = models.UserRole(role)
	user.Active = true
	user.CreatedAt = createdAt
	return user, nil
}

func (s *Store) RevokeToken(ctx context.Context, rawToken string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_tokens WHERE token_hash = ?`, hashToken(rawToken))
	if err != nil {
		return fmt.Errorf("revoca token: %w", err)
	}
	return nil
}

func (s *Store) DeleteExpiredTokens(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_tokens WHERE expires_at <= ?`, formatTime(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("pulizia token scaduti: %w", err)
	}
	return nil
}

func hashToken(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

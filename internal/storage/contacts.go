package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/open-services-lab/customer-registry-api/internal/models"
)

func (s *Store) ListContacts(ctx context.Context, clientID int64, filter ContactFilter) ([]models.Contact, int, error) {
	where := []string{"client_id = ?"}
	args := []any{clientID}

	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + strings.ToLower(query) + "%"
		where = append(where, `(LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(role) LIKE ?)`)
		args = append(args, like, like, like, like)
	}

	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM contacts WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("conteggio referenti: %w", err)
	}

	queryArgs := append(append([]any{}, args...), filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, first_name, last_name, email, role, created_at, updated_at
		FROM contacts
		WHERE `+whereSQL+`
		ORDER BY last_name COLLATE NOCASE ASC, first_name COLLATE NOCASE ASC, id ASC
		LIMIT ? OFFSET ?
	`, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("lista referenti: %w", err)
	}
	defer rows.Close()

	contacts := make([]models.Contact, 0, filter.PageSize)
	for rows.Next() {
		contact, err := scanContact(rows)
		if err != nil {
			return nil, 0, err
		}
		contacts = append(contacts, contact)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("lettura lista referenti: %w", err)
	}
	return contacts, total, nil
}

func (s *Store) GetContact(ctx context.Context, clientID, contactID int64) (models.Contact, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client_id, first_name, last_name, email, role, created_at, updated_at
		FROM contacts
		WHERE id = ? AND client_id = ?
	`, contactID, clientID)
	contact, err := scanContact(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Contact{}, ErrNotFound
		}
		return models.Contact{}, err
	}
	return contact, nil
}

func (s *Store) CreateContact(ctx context.Context, clientID int64, input models.ContactInput) (models.Contact, error) {
	if _, err := s.GetClient(ctx, clientID); err != nil {
		return models.Contact{}, err
	}

	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO contacts (client_id, first_name, last_name, email, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, clientID, input.FirstName, input.LastName, input.Email, input.Role, formatTime(now), formatTime(now))
	if err != nil {
		if isForeignKeyConstraint(err) {
			return models.Contact{}, ErrNotFound
		}
		return models.Contact{}, fmt.Errorf("creazione referente: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Contact{}, fmt.Errorf("id referente creato: %w", err)
	}
	return s.GetContact(ctx, clientID, id)
}

func (s *Store) UpdateContact(ctx context.Context, clientID, contactID int64, input models.ContactInput) (models.Contact, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE contacts
		SET first_name = ?, last_name = ?, email = ?, role = ?, updated_at = ?
		WHERE id = ? AND client_id = ?
	`, input.FirstName, input.LastName, input.Email, input.Role, formatTime(time.Now().UTC()), contactID, clientID)
	if err != nil {
		return models.Contact{}, fmt.Errorf("aggiornamento referente: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return models.Contact{}, fmt.Errorf("righe referente aggiornate: %w", err)
	}
	if affected == 0 {
		return models.Contact{}, ErrNotFound
	}
	return s.GetContact(ctx, clientID, contactID)
}

func (s *Store) DeleteContact(ctx context.Context, clientID, contactID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM contacts WHERE id = ? AND client_id = ?`, contactID, clientID)
	if err != nil {
		return fmt.Errorf("cancellazione referente: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("righe referente cancellate: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func scanContact(scanner rowScanner) (models.Contact, error) {
	var contact models.Contact
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&contact.ID,
		&contact.ClientID,
		&contact.FirstName,
		&contact.LastName,
		&contact.Email,
		&contact.Role,
		&createdAt,
		&updatedAt,
	); err != nil {
		return models.Contact{}, err
	}

	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return models.Contact{}, err
	}
	parsedUpdatedAt, err := parseTime(updatedAt)
	if err != nil {
		return models.Contact{}, err
	}
	contact.CreatedAt = parsedCreatedAt
	contact.UpdatedAt = parsedUpdatedAt
	return contact, nil
}

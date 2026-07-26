package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/open-services-lab/customer-registry-api/internal/models"
)

func (s *Store) ListClients(ctx context.Context, filter ClientFilter) ([]models.Client, int, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 8)

	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + strings.ToLower(query) + "%"
		where = append(where, `(LOWER(business_name) LIKE ? OR LOWER(email) LIKE ? OR vat_number LIKE ?)`)
		args = append(args, like, like, "%"+query+"%")
	}
	if vatNumber := strings.TrimSpace(filter.VATNumber); vatNumber != "" {
		where = append(where, `vat_number = ?`)
		args = append(args, vatNumber)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		where = append(where, `status = ?`)
		args = append(args, status)
	}

	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clients WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("conteggio clienti: %w", err)
	}

	queryArgs := append(append([]any{}, args...), filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, business_name, vat_number, email, phone, address, notes, status, created_at, updated_at
		FROM clients
		WHERE `+whereSQL+`
		ORDER BY business_name COLLATE NOCASE ASC, id ASC
		LIMIT ? OFFSET ?
	`, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("lista clienti: %w", err)
	}
	defer rows.Close()

	clients := make([]models.Client, 0, filter.PageSize)
	for rows.Next() {
		client, err := scanClient(rows)
		if err != nil {
			return nil, 0, err
		}
		clients = append(clients, client)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("lettura lista clienti: %w", err)
	}
	return clients, total, nil
}

func (s *Store) GetClient(ctx context.Context, id int64) (models.Client, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, business_name, vat_number, email, phone, address, notes, status, created_at, updated_at
		FROM clients
		WHERE id = ?
	`, id)
	client, err := scanClient(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Client{}, ErrNotFound
		}
		return models.Client{}, err
	}
	return client, nil
}

func (s *Store) CreateClient(ctx context.Context, input models.ClientInput) (models.Client, error) {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO clients (
			business_name, vat_number, email, phone, address, notes, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		input.BusinessName,
		input.VATNumber,
		input.Email,
		input.Phone,
		input.Address,
		input.Notes,
		input.Status,
		formatTime(now),
		formatTime(now),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return models.Client{}, ErrConflict
		}
		return models.Client{}, fmt.Errorf("creazione cliente: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Client{}, fmt.Errorf("id cliente creato: %w", err)
	}
	return s.GetClient(ctx, id)
}

func (s *Store) UpdateClient(ctx context.Context, id int64, input models.ClientInput) (models.Client, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE clients
		SET business_name = ?, vat_number = ?, email = ?, phone = ?, address = ?, notes = ?, status = ?, updated_at = ?
		WHERE id = ?
	`,
		input.BusinessName,
		input.VATNumber,
		input.Email,
		input.Phone,
		input.Address,
		input.Notes,
		input.Status,
		formatTime(time.Now().UTC()),
		id,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return models.Client{}, ErrConflict
		}
		return models.Client{}, fmt.Errorf("aggiornamento cliente: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return models.Client{}, fmt.Errorf("righe cliente aggiornate: %w", err)
	}
	if affected == 0 {
		return models.Client{}, ErrNotFound
	}
	return s.GetClient(ctx, id)
}

func (s *Store) DeleteClient(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM clients WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("cancellazione cliente: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("righe cliente cancellate: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanClient(scanner rowScanner) (models.Client, error) {
	var client models.Client
	var status string
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(
		&client.ID,
		&client.BusinessName,
		&client.VATNumber,
		&client.Email,
		&client.Phone,
		&client.Address,
		&client.Notes,
		&status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return models.Client{}, err
	}

	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return models.Client{}, err
	}
	parsedUpdatedAt, err := parseTime(updatedAt)
	if err != nil {
		return models.Client{}, err
	}
	client.Status = models.ClientStatus(status)
	client.CreatedAt = parsedCreatedAt
	client.UpdatedAt = parsedUpdatedAt
	return client, nil
}

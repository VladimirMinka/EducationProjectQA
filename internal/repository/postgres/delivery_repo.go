package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"awesomeProject/internal/repository"
)

type DeliveryRepository struct {
	db *sql.DB
}

func NewDeliveryRepository(db *sql.DB) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

func (r *DeliveryRepository) ListAddresses(userID string) ([]repository.Address, error) {
	const q = `
		SELECT id, user_id, title, city, street, building, apartment, postal_code,
		       recipient_name, phone, is_default, created_at
		FROM user_addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC
	`
	rows, err := r.db.QueryContext(context.Background(), q, userID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	defer rows.Close()

	out := make([]repository.Address, 0)
	for rows.Next() {
		addr, err := scanAddress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, addr)
	}
	return out, rows.Err()
}

func (r *DeliveryRepository) GetAddress(userID, addressID string) (repository.Address, error) {
	const q = `
		SELECT id, user_id, title, city, street, building, apartment, postal_code,
		       recipient_name, phone, is_default, created_at
		FROM user_addresses
		WHERE id = $1 AND user_id = $2
	`
	row := r.db.QueryRowContext(context.Background(), q, addressID, userID)
	addr, err := scanAddress(row)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.Address{}, repository.ErrAddressNotFound
	}
	if err != nil {
		return repository.Address{}, fmt.Errorf("get address: %w", err)
	}
	return addr, nil
}

func (r *DeliveryRepository) CreateAddress(addr repository.Address) (repository.Address, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Address{}, fmt.Errorf("begin create address: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if addr.IsDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE user_addresses SET is_default = FALSE WHERE user_id = $1`, addr.UserID); err != nil {
			return repository.Address{}, fmt.Errorf("clear default address: %w", err)
		}
	} else {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_addresses WHERE user_id = $1`, addr.UserID).Scan(&count); err != nil {
			return repository.Address{}, err
		}
		if count == 0 {
			addr.IsDefault = true
		}
	}

	addr.ID = uuid.NewString()
	addr.CreatedAt = time.Now()
	const insert = `
		INSERT INTO user_addresses (
			id, user_id, title, city, street, building, apartment, postal_code,
			recipient_name, phone, is_default, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`
	if _, err := tx.ExecContext(ctx, insert,
		addr.ID, addr.UserID, addr.Title, addr.City, addr.Street, addr.Building, addr.Apartment,
		addr.PostalCode, addr.RecipientName, addr.Phone, addr.IsDefault, addr.CreatedAt,
	); err != nil {
		return repository.Address{}, fmt.Errorf("insert address: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return repository.Address{}, err
	}
	return addr, nil
}

func (r *DeliveryRepository) UpdateAddress(addr repository.Address) (repository.Address, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Address{}, fmt.Errorf("begin update address: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := r.getAddressTx(ctx, tx, addr.UserID, addr.ID)
	if err != nil {
		return repository.Address{}, err
	}

	if addr.IsDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE user_addresses SET is_default = FALSE WHERE user_id = $1`, addr.UserID); err != nil {
			return repository.Address{}, fmt.Errorf("clear default address: %w", err)
		}
	}

	const q = `
		UPDATE user_addresses SET
			title = $3, city = $4, street = $5, building = $6, apartment = $7,
			postal_code = $8, recipient_name = $9, phone = $10, is_default = $11
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, title, city, street, building, apartment, postal_code,
		          recipient_name, phone, is_default, created_at
	`
	row := tx.QueryRowContext(ctx, q,
		addr.ID, addr.UserID, addr.Title, addr.City, addr.Street, addr.Building, addr.Apartment,
		addr.PostalCode, addr.RecipientName, addr.Phone, addr.IsDefault,
	)
	updated, err := scanAddress(row)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.Address{}, repository.ErrAddressNotFound
	}
	if err != nil {
		return repository.Address{}, fmt.Errorf("update address: %w", err)
	}

	if existing.IsDefault && !updated.IsDefault {
		var otherID string
		err := tx.QueryRowContext(ctx, `
			SELECT id FROM user_addresses WHERE user_id = $1 AND id <> $2
			ORDER BY created_at DESC LIMIT 1
		`, addr.UserID, addr.ID).Scan(&otherID)
		if err == nil {
			_, _ = tx.ExecContext(ctx, `UPDATE user_addresses SET is_default = TRUE WHERE id = $1`, otherID)
			updated.IsDefault = false
		} else if errors.Is(err, sql.ErrNoRows) {
			_, _ = tx.ExecContext(ctx, `UPDATE user_addresses SET is_default = TRUE WHERE id = $1`, addr.ID)
			updated.IsDefault = true
		} else if err != nil {
			return repository.Address{}, err
		}
		if updated.IsDefault || otherID != "" {
			fresh, getErr := r.getAddressTx(ctx, tx, addr.UserID, addr.ID)
			if getErr != nil {
				return repository.Address{}, getErr
			}
			updated = fresh
		}
	}

	if err := tx.Commit(); err != nil {
		return repository.Address{}, err
	}
	return updated, nil
}

func (r *DeliveryRepository) DeleteAddress(userID, addressID string) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete address: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := r.getAddressTx(ctx, tx, userID, addressID)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM user_addresses WHERE id = $1 AND user_id = $2`, addressID, userID)
	if err != nil {
		return fmt.Errorf("delete address: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return repository.ErrAddressNotFound
	}

	if existing.IsDefault {
		_, _ = tx.ExecContext(ctx, `
			UPDATE user_addresses SET is_default = TRUE
			WHERE id = (
				SELECT id FROM user_addresses WHERE user_id = $1
				ORDER BY created_at DESC LIMIT 1
			)
		`, userID)
	}

	return tx.Commit()
}

func (r *DeliveryRepository) ListActivePickupPoints() ([]repository.PickupPoint, error) {
	const q = `
		SELECT id, code, city, address, work_hours, active
		FROM pickup_points
		WHERE active = TRUE
		ORDER BY city, code
	`
	rows, err := r.db.QueryContext(context.Background(), q)
	if err != nil {
		return nil, fmt.Errorf("list pickup points: %w", err)
	}
	defer rows.Close()

	out := make([]repository.PickupPoint, 0)
	for rows.Next() {
		var p repository.PickupPoint
		if err := rows.Scan(&p.ID, &p.Code, &p.City, &p.Address, &p.WorkHours, &p.Active); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *DeliveryRepository) GetPickupPoint(id string) (repository.PickupPoint, error) {
	const q = `
		SELECT id, code, city, address, work_hours, active
		FROM pickup_points
		WHERE id = $1
	`
	var p repository.PickupPoint
	err := r.db.QueryRowContext(context.Background(), q, id).Scan(
		&p.ID, &p.Code, &p.City, &p.Address, &p.WorkHours, &p.Active,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.PickupPoint{}, repository.ErrPickupPointNotFound
	}
	if err != nil {
		return repository.PickupPoint{}, fmt.Errorf("get pickup point: %w", err)
	}
	return p, nil
}

func (r *DeliveryRepository) getAddressTx(ctx context.Context, tx *sql.Tx, userID, addressID string) (repository.Address, error) {
	const q = `
		SELECT id, user_id, title, city, street, building, apartment, postal_code,
		       recipient_name, phone, is_default, created_at
		FROM user_addresses
		WHERE id = $1 AND user_id = $2
	`
	addr, err := scanAddress(tx.QueryRowContext(ctx, q, addressID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return repository.Address{}, repository.ErrAddressNotFound
	}
	if err != nil {
		return repository.Address{}, err
	}
	return addr, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAddress(row rowScanner) (repository.Address, error) {
	var addr repository.Address
	err := row.Scan(
		&addr.ID, &addr.UserID, &addr.Title, &addr.City, &addr.Street, &addr.Building, &addr.Apartment,
		&addr.PostalCode, &addr.RecipientName, &addr.Phone, &addr.IsDefault, &addr.CreatedAt,
	)
	return addr, err
}

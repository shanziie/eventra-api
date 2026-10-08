package repository

import (
	"context"
	"errors"
	"fmt"

	"eventra-api/app/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository interface {
	FindByRegistrationID(ctx context.Context, registrationID int) (*model.Payment, error)
	Submit(ctx context.Context, registrationID int, method, referenceNumber string) (*model.Payment, error)
	Verify(ctx context.Context, registrationID int, status string, note string, verifierID int) (*model.Payment, error)
}

type paymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) FindByRegistrationID(ctx context.Context, registrationID int) (*model.Payment, error) {
	query := `
		SELECT id, registration_id, amount, method, reference_number, status, note, submitted_at, verified_at, verified_by, created_at
		FROM payments
		WHERE registration_id = $1
	`
	p := &model.Payment{}
	err := r.db.QueryRow(ctx, query, registrationID).Scan(
		&p.ID,
		&p.RegistrationID,
		&p.Amount,
		&p.Method,
		&p.ReferenceNumber,
		&p.Status,
		&p.Note,
		&p.SubmittedAt,
		&p.VerifiedAt,
		&p.VerifiedBy,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("payment repository find by registration id: %w", TranslateError(err))
	}
	return p, nil
}

func (r *paymentRepository) Submit(ctx context.Context, registrationID int, method, referenceNumber string) (*model.Payment, error) {
	query := `
		UPDATE payments
		SET method = $1, reference_number = $2, status = 'submitted', submitted_at = NOW()
		WHERE registration_id = $3
		RETURNING id, registration_id, amount, method, reference_number, status, note, submitted_at, verified_at, verified_by, created_at
	`
	p := &model.Payment{}
	err := r.db.QueryRow(ctx, query, method, referenceNumber, registrationID).Scan(
		&p.ID,
		&p.RegistrationID,
		&p.Amount,
		&p.Method,
		&p.ReferenceNumber,
		&p.Status,
		&p.Note,
		&p.SubmittedAt,
		&p.VerifiedAt,
		&p.VerifiedBy,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("payment repository submit: %w", TranslateError(err))
	}
	return p, nil
}

func (r *paymentRepository) Verify(ctx context.Context, registrationID int, status string, note string, verifierID int) (*model.Payment, error) {
	query := `
		UPDATE payments
		SET status = $1, note = $2, verified_at = NOW(), verified_by = $3
		WHERE registration_id = $4
		RETURNING id, registration_id, amount, method, reference_number, status, note, submitted_at, verified_at, verified_by, created_at
	`
	p := &model.Payment{}
	err := r.db.QueryRow(ctx, query, status, note, verifierID, registrationID).Scan(
		&p.ID,
		&p.RegistrationID,
		&p.Amount,
		&p.Method,
		&p.ReferenceNumber,
		&p.Status,
		&p.Note,
		&p.SubmittedAt,
		&p.VerifiedAt,
		&p.VerifiedBy,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("payment repository verify: %w", TranslateError(err))
	}
	return p, nil
}

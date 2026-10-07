package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"eventra-api/app/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TicketTypeRepository interface {
	Create(ctx context.Context, eventID int, req *model.CreateTicketTypeRequest) (*model.TicketType, error)
	FindByID(ctx context.Context, id int) (*model.TicketType, error)
	FindByEventID(ctx context.Context, eventID int) ([]model.TicketType, error)
	Update(ctx context.Context, id int, req *model.PutTicketTypeRequest) (*model.TicketType, error)
	Delete(ctx context.Context, id int) error
	SumQuotaByEventID(ctx context.Context, eventID int) (int, error)
	CountRegistrations(ctx context.Context, ticketTypeID int) (int, error)
}

type ticketTypeRepository struct {
	db *pgxpool.Pool
}

func NewTicketTypeRepository(db *pgxpool.Pool) TicketTypeRepository {
	return &ticketTypeRepository{db: db}
}

func (r *ticketTypeRepository) Create(ctx context.Context, eventID int, req *model.CreateTicketTypeRequest) (*model.TicketType, error) {
	query := `
		INSERT INTO ticket_types (event_id, name, price, quota, sold)
		VALUES ($1, $2, $3, $4, 0)
		RETURNING id, event_id, name, price, quota, sold, created_at
	`
	t := &model.TicketType{}
	err := r.db.QueryRow(ctx, query, eventID, req.Name, req.Price, req.Quota).Scan(
		&t.ID,
		&t.EventID,
		&t.Name,
		&t.Price,
		&t.Quota,
		&t.Sold,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ticket type repository create: %w", TranslateError(err))
	}
	return t, nil
}

func (r *ticketTypeRepository) FindByID(ctx context.Context, id int) (*model.TicketType, error) {
	query := `
		SELECT id, event_id, name, price, quota, sold, created_at
		FROM ticket_types
		WHERE id = $1
	`
	t := &model.TicketType{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID,
		&t.EventID,
		&t.Name,
		&t.Price,
		&t.Quota,
		&t.Sold,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("ticket type repository find by id: %w", TranslateError(err))
	}
	return t, nil
}

func (r *ticketTypeRepository) FindByEventID(ctx context.Context, eventID int) ([]model.TicketType, error) {
	query := `
		SELECT id, event_id, name, price, quota, sold, created_at
		FROM ticket_types
		WHERE event_id = $1
		ORDER BY id ASC
	`
	rows, err := r.db.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("ticket type repository find by event id: %w", TranslateError(err))
	}
	defer rows.Close()

	var types []model.TicketType
	for rows.Next() {
		var t model.TicketType
		if err := rows.Scan(&t.ID, &t.EventID, &t.Name, &t.Price, &t.Quota, &t.Sold, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("ticket type scan: %w", TranslateError(err))
		}
		types = append(types, t)
	}
	if types == nil {
		types = []model.TicketType{}
	}
	return types, nil
}

func (r *ticketTypeRepository) Update(ctx context.Context, id int, req *model.PutTicketTypeRequest) (*model.TicketType, error) {
	query := `
		UPDATE ticket_types
		SET name = $1, price = $2, quota = $3
		WHERE id = $4
		RETURNING id, event_id, name, price, quota, sold, created_at
	`
	t := &model.TicketType{}
	err := r.db.QueryRow(ctx, query, req.Name, req.Price, req.Quota, id).Scan(
		&t.ID,
		&t.EventID,
		&t.Name,
		&t.Price,
		&t.Quota,
		&t.Sold,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("ticket type repository update: %w", TranslateError(err))
	}
	return t, nil
}

func (r *ticketTypeRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM ticket_types WHERE id = $1`
	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ticket type repository delete: %w", TranslateError(err))
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ticketTypeRepository) SumQuotaByEventID(ctx context.Context, eventID int) (int, error) {
	query := `SELECT COALESCE(SUM(quota), 0) FROM ticket_types WHERE event_id = $1`
	var sum int
	err := r.db.QueryRow(ctx, query, eventID).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("ticket type repository sum quota: %w", TranslateError(err))
	}
	return sum, nil
}

func (r *ticketTypeRepository) CountRegistrations(ctx context.Context, ticketTypeID int) (int, error) {
	query := `SELECT COUNT(*) FROM registrations WHERE ticket_type_id = $1 AND status <> 'cancelled'`
	var count int
	err := r.db.QueryRow(ctx, query, ticketTypeID).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "relation \"registrations\" does not exist") {
			return 0, nil
		}
		return 0, fmt.Errorf("ticket type repository count registrations: %w", TranslateError(err))
	}
	return count, nil
}

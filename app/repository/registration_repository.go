package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"eventra-api/app/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegistrationRepository interface {
	Create(ctx context.Context, userID, eventID, ticketTypeID int, priceAtPurchase int64) (*model.Registration, error)
	FindByID(ctx context.Context, id int) (*model.Registration, error)
	FindByEventID(ctx context.Context, eventID int, limit int, cursorTime *time.Time, cursorID *int, status string) ([]model.Registration, *string, bool, error)
	FindByUserID(ctx context.Context, userID int, limit int, cursorTime *time.Time, cursorID *int, status string) ([]model.Registration, *string, bool, error)
	FindAllForExport(ctx context.Context, eventID int, status string) ([]model.Registration, error)
	UpdateStatus(ctx context.Context, id int, status string, ticketCode *string) (*model.Registration, error)
	Cancel(ctx context.Context, id int) (*model.Registration, error)
	CheckIn(ctx context.Context, id int) (*model.Registration, error)
	IncrementTicketSold(ctx context.Context, tx pgx.Tx, ticketTypeID int) error
	DecrementTicketSold(ctx context.Context, tx pgx.Tx, ticketTypeID int) error
}

type registrationRepository struct {
	db *pgxpool.Pool
}

func NewRegistrationRepository(db *pgxpool.Pool) RegistrationRepository {
	return &registrationRepository{db: db}
}

func (r *registrationRepository) Create(ctx context.Context, userID, eventID, ticketTypeID int, priceAtPurchase int64) (*model.Registration, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", TranslateError(err))
	}
	defer tx.Rollback(ctx)

	// 1. Lock event & ticket type dengan FOR UPDATE untuk mencegah race condition / oversell
	var capacity, sold, quota int
	err = tx.QueryRow(ctx, `SELECT capacity FROM events WHERE id = $1 FOR UPDATE`, eventID).Scan(&capacity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock event: %w", TranslateError(err))
	}

	err = tx.QueryRow(ctx, `SELECT quota, sold FROM ticket_types WHERE id = $1 AND event_id = $2 FOR UPDATE`, ticketTypeID, eventID).Scan(&quota, &sold)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock ticket type: %w", TranslateError(err))
	}

	// Cek kuota
	if sold >= quota {
		return nil, ErrQuotaFull
	}

	// 2. Insert registrasi baru (status awal 'pending')
	query := `
		INSERT INTO registrations (user_id, event_id, ticket_type_id, status, price_at_purchase)
		VALUES ($1, $2, $3, 'pending', $4)
		RETURNING id, user_id, event_id, ticket_type_id, status, price_at_purchase, ticket_code, confirmed_at, checked_in_at, cancelled_at, created_at, updated_at
	`
	reg := &model.Registration{}
	err = tx.QueryRow(ctx, query, userID, eventID, ticketTypeID, priceAtPurchase).Scan(
		&reg.ID,
		&reg.UserID,
		&reg.EventID,
		&reg.TicketTypeID,
		&reg.Status,
		&reg.PriceAtPurchase,
		&reg.TicketCode,
		&reg.ConfirmedAt,
		&reg.CheckedInAt,
		&reg.CancelledAt,
		&reg.CreatedAt,
		&reg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert registration: %w", TranslateError(err))
	}

	// 3. Increment sold pada ticket_types
	if err := r.IncrementTicketSold(ctx, tx, ticketTypeID); err != nil {
		return nil, err
	}

	// 4. Buat record payment dengan status 'unpaid'
	paymentQuery := `
		INSERT INTO payments (registration_id, amount, status, note)
		VALUES ($1, $2, 'unpaid', '')
	`
	if _, err := tx.Exec(ctx, paymentQuery, reg.ID, priceAtPurchase); err != nil {
		return nil, fmt.Errorf("insert payment: %w", TranslateError(err))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", TranslateError(err))
	}

	return reg, nil
}

func (r *registrationRepository) FindByID(ctx context.Context, id int) (*model.Registration, error) {
	query := `
		SELECT r.id, r.user_id, r.event_id, r.ticket_type_id, r.status, r.price_at_purchase, r.ticket_code, r.confirmed_at, r.checked_in_at, r.cancelled_at, r.created_at, r.updated_at,
		       u.username, u.email, u.full_name, e.title, t.name
		FROM registrations r
		JOIN users u ON r.user_id = u.id
		JOIN events e ON r.event_id = e.id
		JOIN ticket_types t ON r.ticket_type_id = t.id
		WHERE r.id = $1
	`
	reg := &model.Registration{}
	var username, email, fullName, eventTitle, ticketName string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&reg.ID,
		&reg.UserID,
		&reg.EventID,
		&reg.TicketTypeID,
		&reg.Status,
		&reg.PriceAtPurchase,
		&reg.TicketCode,
		&reg.ConfirmedAt,
		&reg.CheckedInAt,
		&reg.CancelledAt,
		&reg.CreatedAt,
		&reg.UpdatedAt,
		&username,
		&email,
		&fullName,
		&eventTitle,
		&ticketName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("registration repository find by id: %w", TranslateError(err))
	}
	reg.Username = &username
	reg.Email = &email
	reg.FullName = &fullName
	reg.EventTitle = &eventTitle
	reg.TicketName = &ticketName
	return reg, nil
}

func (r *registrationRepository) FindByEventID(ctx context.Context, eventID int, limit int, cursorTime *time.Time, cursorID *int, status string) ([]model.Registration, *string, bool, error) {
	whereClauses := []string{"r.event_id = $1"}
	args := []any{eventID}
	argCount := 2

	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("r.status = $%d", argCount))
		args = append(args, status)
		argCount++
	}

	if cursorTime != nil && cursorID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(r.created_at, r.id) < ($%d, $%d)", argCount, argCount+1))
		args = append(args, *cursorTime, *cursorID)
		argCount += 2
	}

	whereStr := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT r.id, r.user_id, r.event_id, r.ticket_type_id, r.status, r.price_at_purchase, r.ticket_code, r.confirmed_at, r.checked_in_at, r.cancelled_at, r.created_at, r.updated_at,
		       u.username, u.email, u.full_name, e.title, t.name
		FROM registrations r
		JOIN users u ON r.user_id = u.id
		JOIN events e ON r.event_id = e.id
		JOIN ticket_types t ON r.ticket_type_id = t.id
		WHERE %s
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $%d
	`, whereStr, argCount)

	args = append(args, limit+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, false, fmt.Errorf("find registrations by event id: %w", TranslateError(err))
	}
	defer rows.Close()

	var regs []model.Registration
	for rows.Next() {
		var reg model.Registration
		var username, email, fullName, eventTitle, ticketName string
		if err := rows.Scan(
			&reg.ID,
			&reg.UserID,
			&reg.EventID,
			&reg.TicketTypeID,
			&reg.Status,
			&reg.PriceAtPurchase,
			&reg.TicketCode,
			&reg.ConfirmedAt,
			&reg.CheckedInAt,
			&reg.CancelledAt,
			&reg.CreatedAt,
			&reg.UpdatedAt,
			&username,
			&email,
			&fullName,
			&eventTitle,
			&ticketName,
		); err != nil {
			return nil, nil, false, fmt.Errorf("registration scan: %w", TranslateError(err))
		}
		reg.Username = &username
		reg.Email = &email
		reg.FullName = &fullName
		reg.EventTitle = &eventTitle
		reg.TicketName = &ticketName
		regs = append(regs, reg)
	}

	hasMore := false
	if len(regs) > limit {
		hasMore = true
		regs = regs[:limit]
	}

	var nextCursor *string
	if hasMore && len(regs) > 0 {
		last := regs[len(regs)-1]
		cStr := fmt.Sprintf("%d|%d", last.CreatedAt.UnixNano(), last.ID)
		nextCursor = &cStr
	}

	if regs == nil {
		regs = []model.Registration{}
	}

	return regs, nextCursor, hasMore, nil
}

func (r *registrationRepository) FindByUserID(ctx context.Context, userID int, limit int, cursorTime *time.Time, cursorID *int, status string) ([]model.Registration, *string, bool, error) {
	whereClauses := []string{"r.user_id = $1"}
	args := []any{userID}
	argCount := 2

	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("r.status = $%d", argCount))
		args = append(args, status)
		argCount++
	}

	if cursorTime != nil && cursorID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(r.created_at, r.id) < ($%d, $%d)", argCount, argCount+1))
		args = append(args, *cursorTime, *cursorID)
		argCount += 2
	}

	whereStr := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT r.id, r.user_id, r.event_id, r.ticket_type_id, r.status, r.price_at_purchase, r.ticket_code, r.confirmed_at, r.checked_in_at, r.cancelled_at, r.created_at, r.updated_at,
		       u.username, u.email, u.full_name, e.title, t.name
		FROM registrations r
		JOIN users u ON r.user_id = u.id
		JOIN events e ON r.event_id = e.id
		JOIN ticket_types t ON r.ticket_type_id = t.id
		WHERE %s
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $%d
	`, whereStr, argCount)

	args = append(args, limit+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, false, fmt.Errorf("find registrations by user id: %w", TranslateError(err))
	}
	defer rows.Close()

	var regs []model.Registration
	for rows.Next() {
		var reg model.Registration
		var username, email, fullName, eventTitle, ticketName string
		if err := rows.Scan(
			&reg.ID,
			&reg.UserID,
			&reg.EventID,
			&reg.TicketTypeID,
			&reg.Status,
			&reg.PriceAtPurchase,
			&reg.TicketCode,
			&reg.ConfirmedAt,
			&reg.CheckedInAt,
			&reg.CancelledAt,
			&reg.CreatedAt,
			&reg.UpdatedAt,
			&username,
			&email,
			&fullName,
			&eventTitle,
			&ticketName,
		); err != nil {
			return nil, nil, false, fmt.Errorf("registration scan: %w", TranslateError(err))
		}
		reg.Username = &username
		reg.Email = &email
		reg.FullName = &fullName
		reg.EventTitle = &eventTitle
		reg.TicketName = &ticketName
		regs = append(regs, reg)
	}

	hasMore := false
	if len(regs) > limit {
		hasMore = true
		regs = regs[:limit]
	}

	var nextCursor *string
	if hasMore && len(regs) > 0 {
		last := regs[len(regs)-1]
		cStr := fmt.Sprintf("%d|%d", last.CreatedAt.UnixNano(), last.ID)
		nextCursor = &cStr
	}

	if regs == nil {
		regs = []model.Registration{}
	}

	return regs, nextCursor, hasMore, nil
}

func (r *registrationRepository) FindAllForExport(ctx context.Context, eventID int, status string) ([]model.Registration, error) {
	whereClauses := []string{"r.event_id = $1"}
	args := []any{eventID}
	argCount := 2

	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("r.status = $%d", argCount))
		args = append(args, status)
		argCount++
	}

	query := fmt.Sprintf(`
		SELECT r.id, r.user_id, r.event_id, r.ticket_type_id, r.status, r.price_at_purchase, r.ticket_code, r.confirmed_at, r.checked_in_at, r.cancelled_at, r.created_at, r.updated_at,
		       u.username, u.email, u.full_name, e.title, t.name
		FROM registrations r
		JOIN users u ON r.user_id = u.id
		JOIN events e ON r.event_id = e.id
		JOIN ticket_types t ON r.ticket_type_id = t.id
		WHERE %s
		ORDER BY r.created_at DESC, r.id DESC
	`, strings.Join(whereClauses, " AND "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find registrations for export: %w", TranslateError(err))
	}
	defer rows.Close()

	var regs []model.Registration
	for rows.Next() {
		var reg model.Registration
		var username, email, fullName, eventTitle, ticketName string
		if err := rows.Scan(
			&reg.ID,
			&reg.UserID,
			&reg.EventID,
			&reg.TicketTypeID,
			&reg.Status,
			&reg.PriceAtPurchase,
			&reg.TicketCode,
			&reg.ConfirmedAt,
			&reg.CheckedInAt,
			&reg.CancelledAt,
			&reg.CreatedAt,
			&reg.UpdatedAt,
			&username,
			&email,
			&fullName,
			&eventTitle,
			&ticketName,
		); err != nil {
			return nil, fmt.Errorf("registration scan: %w", TranslateError(err))
		}
		reg.Username = &username
		reg.Email = &email
		reg.FullName = &fullName
		reg.EventTitle = &eventTitle
		reg.TicketName = &ticketName
		regs = append(regs, reg)
	}
	if regs == nil {
		regs = []model.Registration{}
	}
	return regs, nil
}

func (r *registrationRepository) UpdateStatus(ctx context.Context, id int, status string, ticketCode *string) (*model.Registration, error) {
	query := `
		UPDATE registrations
		SET status = $1, ticket_code = COALESCE($2, ticket_code), confirmed_at = CASE WHEN $1 = 'confirmed' THEN NOW() ELSE confirmed_at END, updated_at = NOW()
		WHERE id = $3
		RETURNING id, user_id, event_id, ticket_type_id, status, price_at_purchase, ticket_code, confirmed_at, checked_in_at, cancelled_at, created_at, updated_at
	`
	reg := &model.Registration{}
	err := r.db.QueryRow(ctx, query, status, ticketCode, id).Scan(
		&reg.ID,
		&reg.UserID,
		&reg.EventID,
		&reg.TicketTypeID,
		&reg.Status,
		&reg.PriceAtPurchase,
		&reg.TicketCode,
		&reg.ConfirmedAt,
		&reg.CheckedInAt,
		&reg.CancelledAt,
		&reg.CreatedAt,
		&reg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update registration status: %w", TranslateError(err))
	}
	return reg, nil
}

func (r *registrationRepository) Cancel(ctx context.Context, id int) (*model.Registration, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", TranslateError(err))
	}
	defer tx.Rollback(ctx)

	var ticketTypeID int
	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT ticket_type_id, status FROM registrations WHERE id = $1 FOR UPDATE`, id).Scan(&ticketTypeID, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock registration: %w", TranslateError(err))
	}

	if currentStatus == "cancelled" {
		return nil, ErrAlreadyCancelled
	}

	query := `
		UPDATE registrations
		SET status = 'cancelled', cancelled_at = NOW(), updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, event_id, ticket_type_id, status, price_at_purchase, ticket_code, confirmed_at, checked_in_at, cancelled_at, created_at, updated_at
	`
	reg := &model.Registration{}
	err = tx.QueryRow(ctx, query, id).Scan(
		&reg.ID,
		&reg.UserID,
		&reg.EventID,
		&reg.TicketTypeID,
		&reg.Status,
		&reg.PriceAtPurchase,
		&reg.TicketCode,
		&reg.ConfirmedAt,
		&reg.CheckedInAt,
		&reg.CancelledAt,
		&reg.CreatedAt,
		&reg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("cancel registration: %w", TranslateError(err))
	}

	// Kurangi sold pada ticket_types jika sebelumnya belum cancelled
	if currentStatus != "cancelled" {
		if err := r.DecrementTicketSold(ctx, tx, ticketTypeID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", TranslateError(err))
	}

	return reg, nil
}

func (r *registrationRepository) CheckIn(ctx context.Context, id int) (*model.Registration, error) {
	query := `
		UPDATE registrations
		SET status = 'checked_in', checked_in_at = NOW(), updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, event_id, ticket_type_id, status, price_at_purchase, ticket_code, confirmed_at, checked_in_at, cancelled_at, created_at, updated_at
	`
	reg := &model.Registration{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&reg.ID,
		&reg.UserID,
		&reg.EventID,
		&reg.TicketTypeID,
		&reg.Status,
		&reg.PriceAtPurchase,
		&reg.TicketCode,
		&reg.ConfirmedAt,
		&reg.CheckedInAt,
		&reg.CancelledAt,
		&reg.CreatedAt,
		&reg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("check in registration: %w", TranslateError(err))
	}
	return reg, nil
}

func (r *registrationRepository) IncrementTicketSold(ctx context.Context, tx pgx.Tx, ticketTypeID int) error {
	_, err := tx.Exec(ctx, `UPDATE ticket_types SET sold = sold + 1 WHERE id = $1`, ticketTypeID)
	if err != nil {
		return fmt.Errorf("increment ticket sold: %w", TranslateError(err))
	}
	return nil
}

func (r *registrationRepository) DecrementTicketSold(ctx context.Context, tx pgx.Tx, ticketTypeID int) error {
	_, err := tx.Exec(ctx, `UPDATE ticket_types SET sold = GREATEST(0, sold - 1) WHERE id = $1`, ticketTypeID)
	if err != nil {
		return fmt.Errorf("decrement ticket sold: %w", TranslateError(err))
	}
	return nil
}

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

type EventRepository interface {
	Create(ctx context.Context, organizerID int, req *model.CreateEventRequest) (*model.Event, error)
	FindByID(ctx context.Context, id int) (*model.Event, error)
	FindAll(ctx context.Context, limit int, cursorTime *time.Time, cursorID *int, search string, categoryID *int, status string, viewerRole string, viewerID int) ([]model.Event, *string, bool, error)
	Update(ctx context.Context, id int, req *model.PutEventRequest) (*model.Event, error)
	Patch(ctx context.Context, id int, fields map[string]any) (*model.Event, error)
	Delete(ctx context.Context, id int) error
	UpdateStatus(ctx context.Context, id int, status string) (*model.Event, error)
	CountActiveRegistrations(ctx context.Context, eventID int) (int, error)
}

type eventRepository struct {
	db *pgxpool.Pool
}

func NewEventRepository(db *pgxpool.Pool) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Create(ctx context.Context, organizerID int, req *model.CreateEventRequest) (*model.Event, error) {
	query := `
		INSERT INTO events (organizer_id, category_id, title, description, location, capacity, price, starts_at, ends_at, registration_deadline, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'draft')
		RETURNING id, organizer_id, category_id, title, description, location, capacity, price, status, starts_at, ends_at, registration_deadline, created_at, updated_at
	`
	e := &model.Event{}
	err := r.db.QueryRow(ctx, query,
		organizerID,
		req.CategoryID,
		req.Title,
		req.Description,
		req.Location,
		req.Capacity,
		req.Price,
		req.StartsAt,
		req.EndsAt,
		req.RegistrationDeadline,
	).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.CategoryID,
		&e.Title,
		&e.Description,
		&e.Location,
		&e.Capacity,
		&e.Price,
		&e.Status,
		&e.StartsAt,
		&e.EndsAt,
		&e.RegistrationDeadline,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("event repository create: %w", TranslateError(err))
	}
	return e, nil
}

func (r *eventRepository) FindByID(ctx context.Context, id int) (*model.Event, error) {
	query := `
		SELECT e.id, e.organizer_id, e.category_id, e.title, e.description, e.location, e.capacity, e.price, e.status, e.starts_at, e.ends_at, e.registration_deadline, e.created_at, e.updated_at,
		       u.username, c.name
		FROM events e
		JOIN users u ON e.organizer_id = u.id
		JOIN categories c ON e.category_id = c.id
		WHERE e.id = $1
	`
	e := &model.Event{}
	var orgName, catName string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.CategoryID,
		&e.Title,
		&e.Description,
		&e.Location,
		&e.Capacity,
		&e.Price,
		&e.Status,
		&e.StartsAt,
		&e.EndsAt,
		&e.RegistrationDeadline,
		&e.CreatedAt,
		&e.UpdatedAt,
		&orgName,
		&catName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("event repository find by id: %w", TranslateError(err))
	}
	e.OrganizerName = &orgName
	e.CategoryName = &catName
	return e, nil
}

func (r *eventRepository) FindAll(ctx context.Context, limit int, cursorTime *time.Time, cursorID *int, search string, categoryID *int, status string, viewerRole string, viewerID int) ([]model.Event, *string, bool, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argCount := 1

	// Visibilitas (BR-E9):
	// participant: hanya published dan finished
	// organizer: published, finished, atau event miliknya sendiri (draft/cancelled)
	// admin: semua
	switch viewerRole {
case "participant":
		whereClauses = append(whereClauses, "e.status IN ('published', 'finished')")
	case "organizer":
		whereClauses = append(whereClauses, fmt.Sprintf("(e.status IN ('published', 'finished') OR e.organizer_id = $%d)", argCount))
		args = append(args, viewerID)
		argCount++
	}
	// admin bebas melihat semua status

	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(e.title ILIKE $%d OR e.description ILIKE $%d OR e.location ILIKE $%d)", argCount, argCount, argCount))
		args = append(args, "%"+search+"%")
		argCount++
	}

	if categoryID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.category_id = $%d", argCount))
		args = append(args, *categoryID)
		argCount++
	}

	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("e.status = $%d", argCount))
		args = append(args, status)
		argCount++
	}

	// Cursor pagination (created_at DESC, id DESC)
	if cursorTime != nil && cursorID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(e.created_at, e.id) < ($%d, $%d)", argCount, argCount+1))
		args = append(args, *cursorTime, *cursorID)
		argCount += 2
	}

	whereStr := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT e.id, e.organizer_id, e.category_id, e.title, e.description, e.location, e.capacity, e.price, e.status, e.starts_at, e.ends_at, e.registration_deadline, e.created_at, e.updated_at,
		       u.username, c.name
		FROM events e
		JOIN users u ON e.organizer_id = u.id
		JOIN categories c ON e.category_id = c.id
		WHERE %s
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $%d
	`, whereStr, argCount)

	// Ambil limit + 1 untuk mendeteksi has_more
	args = append(args, limit+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, false, fmt.Errorf("event repository find all: %w", TranslateError(err))
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var e model.Event
		var orgName, catName string
		if err := rows.Scan(
			&e.ID,
			&e.OrganizerID,
			&e.CategoryID,
			&e.Title,
			&e.Description,
			&e.Location,
			&e.Capacity,
			&e.Price,
			&e.Status,
			&e.StartsAt,
			&e.EndsAt,
			&e.RegistrationDeadline,
			&e.CreatedAt,
			&e.UpdatedAt,
			&orgName,
			&catName,
		); err != nil {
			return nil, nil, false, fmt.Errorf("event scan: %w", TranslateError(err))
		}
		e.OrganizerName = &orgName
		e.CategoryName = &catName
		events = append(events, e)
	}

	hasMore := false
	if len(events) > limit {
		hasMore = true
		events = events[:limit]
	}

	var nextCursor *string
	if hasMore && len(events) > 0 {
		last := events[len(events)-1]
		// Format cursor: base64 atau string gabungan created_at|id
		// Di sini kita gunakan format epoch_nano|id agar mudah di-parse
		cStr := fmt.Sprintf("%d|%d", last.CreatedAt.UnixNano(), last.ID)
		nextCursor = &cStr
	}

	if events == nil {
		events = []model.Event{}
	}

	return events, nextCursor, hasMore, nil
}

func (r *eventRepository) Update(ctx context.Context, id int, req *model.PutEventRequest) (*model.Event, error) {
	query := `
		UPDATE events
		SET category_id = $1, title = $2, description = $3, location = $4, capacity = $5, price = $6, starts_at = $7, ends_at = $8, registration_deadline = $9, updated_at = NOW()
		WHERE id = $10
		RETURNING id, organizer_id, category_id, title, description, location, capacity, price, status, starts_at, ends_at, registration_deadline, created_at, updated_at
	`
	e := &model.Event{}
	err := r.db.QueryRow(ctx, query,
		req.CategoryID,
		req.Title,
		req.Description,
		req.Location,
		req.Capacity,
		req.Price,
		req.StartsAt,
		req.EndsAt,
		req.RegistrationDeadline,
		id,
	).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.CategoryID,
		&e.Title,
		&e.Description,
		&e.Location,
		&e.Capacity,
		&e.Price,
		&e.Status,
		&e.StartsAt,
		&e.EndsAt,
		&e.RegistrationDeadline,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("event repository update: %w", TranslateError(err))
	}
	return e, nil
}

func (r *eventRepository) Patch(ctx context.Context, id int, fields map[string]any) (*model.Event, error) {
	if len(fields) == 0 {
		return r.FindByID(ctx, id)
	}

	sets := []string{}
	args := []any{}
	argCount := 1

	for k, v := range fields {
		sets = append(sets, fmt.Sprintf("%s = $%d", k, argCount))
		args = append(args, v)
		argCount++
	}
	sets = append(sets, "updated_at = NOW()")

	query := fmt.Sprintf(`
		UPDATE events
		SET %s
		WHERE id = $%d
		RETURNING id, organizer_id, category_id, title, description, location, capacity, price, status, starts_at, ends_at, registration_deadline, created_at, updated_at
	`, strings.Join(sets, ", "), argCount)

	args = append(args, id)

	e := &model.Event{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.CategoryID,
		&e.Title,
		&e.Description,
		&e.Location,
		&e.Capacity,
		&e.Price,
		&e.Status,
		&e.StartsAt,
		&e.EndsAt,
		&e.RegistrationDeadline,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("event repository patch: %w", TranslateError(err))
	}
	return e, nil
}

func (r *eventRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM events WHERE id = $1`
	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("event repository delete: %w", TranslateError(err))
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *eventRepository) UpdateStatus(ctx context.Context, id int, status string) (*model.Event, error) {
	query := `
		UPDATE events
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, organizer_id, category_id, title, description, location, capacity, price, status, starts_at, ends_at, registration_deadline, created_at, updated_at
	`
	e := &model.Event{}
	err := r.db.QueryRow(ctx, query, status, id).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.CategoryID,
		&e.Title,
		&e.Description,
		&e.Location,
		&e.Capacity,
		&e.Price,
		&e.Status,
		&e.StartsAt,
		&e.EndsAt,
		&e.RegistrationDeadline,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("event repository update status: %w", TranslateError(err))
	}
	return e, nil
}

func (r *eventRepository) CountActiveRegistrations(ctx context.Context, eventID int) (int, error) {
	// Memeriksa jumlah registrasi aktif (status <> 'cancelled')
	query := `SELECT COUNT(*) FROM registrations WHERE event_id = $1 AND status <> 'cancelled'`
	var count int
	err := r.db.QueryRow(ctx, query, eventID).Scan(&count)
	if err != nil {
		// Jika tabel registrations belum ada atau error lain, tangani atau return 0 jika migration 005 belum tereksekusi
		if strings.Contains(err.Error(), "relation \"registrations\" does not exist") {
			return 0, nil
		}
		return 0, fmt.Errorf("event repository count active registrations: %w", TranslateError(err))
	}
	return count, nil
}

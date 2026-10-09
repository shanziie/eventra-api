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

// UserRepository adalah kontrak akses data pengguna di database.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	FindByID(ctx context.Context, id int) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindAll(ctx context.Context, limit, offset int, search, role string, isActive *bool, sortBy, sortOrder string) ([]model.User, int, error)
	Update(ctx context.Context, id int, req *model.PutUserRequest) (*model.User, error)
	Patch(ctx context.Context, id int, username, email, fullName *string, isActive *bool) (*model.User, error)
	Delete(ctx context.Context, id int) error
	UpdateRole(ctx context.Context, id int, role string) (*model.User, error)
	CountEventsByOrganizer(ctx context.Context, userID int) (int, error)
	CountRegistrationsByUser(ctx context.Context, userID int) (int, error)
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	query := `
		INSERT INTO users (username, email, full_name, password, role, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, username, email, full_name, password, role, is_active, created_at
	`
	created := &model.User{}
	err := r.db.QueryRow(ctx, query,
		user.Username,
		user.Email,
		user.FullName,
		user.Password,
		user.Role,
		user.IsActive,
	).Scan(
		&created.ID,
		&created.Username,
		&created.Email,
		&created.FullName,
		&created.Password,
		&created.Role,
		&created.IsActive,
		&created.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user repository create: %w", TranslateError(err))
	}
	return created, nil
}

func (r *userRepository) FindByID(ctx context.Context, id int) (*model.User, error) {
	query := `
		SELECT id, username, email, full_name, password, role, is_active, created_at
		FROM users
		WHERE id = $1
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.FullName,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user repository find by id: %w", TranslateError(err))
	}
	return u, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, email, full_name, password, role, is_active, created_at
		FROM users
		WHERE LOWER(username) = LOWER($1)
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.FullName,
		&u.Password,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user repository find by username: %w", TranslateError(err))
	}
	return u, nil
}

func (r *userRepository) FindAll(ctx context.Context, limit, offset int, search, role string, isActive *bool, sortBy, sortOrder string) ([]model.User, int, error) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argCount := 1

	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(username ILIKE $%d OR email ILIKE $%d OR full_name ILIKE $%d)", argCount, argCount, argCount))
		args = append(args, "%"+search+"%")
		argCount++
	}

	if role != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("role = $%d", argCount))
		args = append(args, role)
		argCount++
	}

	if isActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_active = $%d", argCount))
		args = append(args, *isActive)
		argCount++
	}

	whereStr := strings.Join(whereClauses, " AND ")

	// Hitung total data
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", whereStr)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("user repository count: %w", TranslateError(err))
	}

	// Whitelist sortBy dan sortOrder
	allowedSort := map[string]string{
		"created_at": "created_at",
		"username":   "username",
		"email":      "email",
		"full_name":  "full_name",
	}
	col := allowedSort[sortBy]
	if col == "" {
		col = "created_at"
	}

	ord := "DESC"
	if strings.ToUpper(sortOrder) == "ASC" {
		ord = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, username, email, full_name, password, role, is_active, created_at
		FROM users
		WHERE %s
		ORDER BY %s %s, id DESC
		LIMIT $%d OFFSET $%d
	`, whereStr, col, ord, argCount, argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("user repository find all: %w", TranslateError(err))
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("user scan: %w", TranslateError(err))
		}
		users = append(users, u)
	}
	if users == nil {
		users = []model.User{}
	}

	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, id int, req *model.PutUserRequest) (*model.User, error) {
	query := `
		UPDATE users
		SET username = $1, email = $2, full_name = $3, is_active = $4
		WHERE id = $5
		RETURNING id, username, email, full_name, password, role, is_active, created_at
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, req.Username, req.Email, req.FullName, *req.IsActive, id).
		Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user repository update: %w", TranslateError(err))
	}
	return u, nil
}

func (r *userRepository) Patch(ctx context.Context, id int, username, email, fullName *string, isActive *bool) (*model.User, error) {
	sets := []string{}
	args := []any{}
	argCount := 1

	if username != nil {
		sets = append(sets, fmt.Sprintf("username = $%d", argCount))
		args = append(args, *username)
		argCount++
	}
	if email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", argCount))
		args = append(args, *email)
		argCount++
	}
	if fullName != nil {
		sets = append(sets, fmt.Sprintf("full_name = $%d", argCount))
		args = append(args, *fullName)
		argCount++
	}
	if isActive != nil {
		sets = append(sets, fmt.Sprintf("is_active = $%d", argCount))
		args = append(args, *isActive)
		argCount++
	}

	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE id = $%d
		RETURNING id, username, email, full_name, password, role, is_active, created_at
	`, strings.Join(sets, ", "), argCount)

	args = append(args, id)

	u := &model.User{}
	err := r.db.QueryRow(ctx, query, args...).
		Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user repository patch: %w", TranslateError(err))
	}
	return u, nil
}

func (r *userRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`
	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user repository delete: %w", TranslateError(err))
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) UpdateRole(ctx context.Context, id int, role string) (*model.User, error) {
	query := `
		UPDATE users
		SET role = $1
		WHERE id = $2
		RETURNING id, username, email, full_name, password, role, is_active, created_at
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, role, id).
		Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user repository update role: %w", TranslateError(err))
	}
	return u, nil
}

func (r *userRepository) CountEventsByOrganizer(ctx context.Context, userID int) (int, error) {
	query := `SELECT COUNT(*) FROM events WHERE organizer_id = $1`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("user repository count events: %w", TranslateError(err))
	}
	return count, nil
}

func (r *userRepository) CountRegistrationsByUser(ctx context.Context, userID int) (int, error) {
	query := `SELECT COUNT(*) FROM registrations WHERE user_id = $1`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("user repository count registrations: %w", TranslateError(err))
	}
	return count, nil
}

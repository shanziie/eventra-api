package repository

import (
	"context"
	"fmt"

	"eventra-api/app/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository adalah kontrak akses data pengguna di database.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	FindByID(ctx context.Context, id int) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
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
		return nil, fmt.Errorf("user repository find by username: %w", TranslateError(err))
	}
	return u, nil
}

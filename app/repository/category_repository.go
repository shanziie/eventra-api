package repository

import (
	"context"
	"errors"
	"fmt"

	"eventra-api/app/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// kontrak akses data kategori event
type CategoryRepository interface {
	FindAll(ctx context.Context) ([]model.Category, error)
	FindByID(ctx context.Context, id int) (*model.Category, error)
	Create(ctx context.Context, req *model.CategoryRequest) (*model.Category, error)
	Update(ctx context.Context, id int, req *model.CategoryRequest) (*model.Category, error)
	Delete(ctx context.Context, id int) error
	CountEvents(ctx context.Context, categoryID int) (int, error)
}

type categoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) FindAll(ctx context.Context) ([]model.Category, error) {
	query := `SELECT id, name, description, created_at FROM categories ORDER BY id ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("category repository find all: %w", TranslateError(err))
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("category scan: %w", TranslateError(err))
		}
		categories = append(categories, c)
	}
	if categories == nil {
		categories = []model.Category{}
	}
	return categories, nil
}

func (r *categoryRepository) FindByID(ctx context.Context, id int) (*model.Category, error) {
	query := `SELECT id, name, description, created_at FROM categories WHERE id = $1`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("category repository find by id: %w", TranslateError(err))
	}
	return c, nil
}

func (r *categoryRepository) Create(ctx context.Context, req *model.CategoryRequest) (*model.Category, error) {
	query := `
		INSERT INTO categories (name, description, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, name, description, created_at
	`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query, req.Name, req.Description).
		Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("category repository create: %w", TranslateError(err))
	}
	return c, nil
}

func (r *categoryRepository) Update(ctx context.Context, id int, req *model.CategoryRequest) (*model.Category, error) {
	query := `
		UPDATE categories
		SET name = $1, description = $2
		WHERE id = $3
		RETURNING id, name, description, created_at
	`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query, req.Name, req.Description, id).
		Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("category repository update: %w", TranslateError(err))
	}
	return c, nil
}

func (r *categoryRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM categories WHERE id = $1`
	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("category repository delete: %w", TranslateError(err))
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *categoryRepository) CountEvents(ctx context.Context, categoryID int) (int, error) {
	query := `SELECT COUNT(*) FROM events WHERE category_id = $1`
	var count int
	err := r.db.QueryRow(ctx, query, categoryID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("category repository count events: %w", TranslateError(err))
	}
	return count, nil
}

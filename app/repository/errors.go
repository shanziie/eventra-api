package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound = errors.New("data tidak ditemukan")
	ErrConflict = errors.New("konflik integritas data")
)

// DuplicateError membawa informasi nama constraint agar service bisa membedakan duplikat username atau email.
type DuplicateError struct {
	Constraint string
}

func (e *DuplicateError) Error() string {
	return "data duplikat: " + e.Constraint
}

// TranslateError menerjemahkan error spesifik pgx/PostgreSQL ke sentinel error repository.
// Error teknis pgx tidak boleh bocor ke layer atas.
func TranslateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return &DuplicateError{Constraint: pgErr.ConstraintName}
		case "23503": // foreign_key_violation
			return ErrConflict
		case "23514": // check_violation
			return ErrConflict
		}
	}

	return err
}

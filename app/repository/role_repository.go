package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepository mengelola akses data role dan relasi hak akses permission.
type RoleRepository interface {
	LoadPermissions(ctx context.Context) (map[string][]string, error)
}

type roleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) RoleRepository {
	return &roleRepository{db: db}
}

// LoadPermissions memuat semua role dan permission dengan LEFT JOIN
// agar role yang belum memiliki permission tetap tercatat sebagai role yang sah.
func (r *roleRepository) LoadPermissions(ctx context.Context) (map[string][]string, error) {
	query := `
		SELECT r.name, COALESCE(rp.permission_name, '')
		FROM roles r
		LEFT JOIN role_permissions rp ON r.name = rp.role_name
		ORDER BY r.name
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("role repository load permissions: %w", TranslateError(err))
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var roleName, permName string
		if err := rows.Scan(&roleName, &permName); err != nil {
			return nil, fmt.Errorf("role repository scan: %w", TranslateError(err))
		}
		if _, exists := result[roleName]; !exists {
			result[roleName] = []string{}
		}
		if permName != "" {
			result[roleName] = append(result[roleName], permName)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("role repository rows: %w", TranslateError(err))
	}

	return result, nil
}

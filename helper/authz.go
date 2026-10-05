package helper

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	permMu          sync.RWMutex
	rolePermissions = make(map[string]map[string]bool)
)

// LoadPermissions memuat seluruh pasangan role_permissions dari database ke memori sekali saat startup.
func LoadPermissions(ctx context.Context, db *pgxpool.Pool) error {
	query := `SELECT role_name, permission_name FROM role_permissions`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("load permissions query: %w", err)
	}
	defer rows.Close()

	temp := make(map[string]map[string]bool)
	for rows.Next() {
		var role, perm string
		if err := rows.Scan(&role, &perm); err != nil {
			return fmt.Errorf("load permissions scan: %w", err)
		}
		if temp[role] == nil {
			temp[role] = make(map[string]bool)
		}
		temp[role][perm] = true
	}
	if rows.Err() != nil {
		return fmt.Errorf("load permissions rows: %w", rows.Err())
	}

	permMu.Lock()
	rolePermissions = temp
	permMu.Unlock()
	return nil
}

// Can memeriksa apakah suatu role memiliki permission (fail closed: false jika role/perm tidak ada).
func Can(role, permission string) bool {
	permMu.RLock()
	defer permMu.RUnlock()

	perms, exists := rolePermissions[role]
	if !exists {
		return false
	}
	return perms[permission]
}

// GetPermissionsByRole mengembalikan daftar permission untuk suatu role (misalnya untuk endpoint /auth/me).
func GetPermissionsByRole(role string) []string {
	permMu.RLock()
	defer permMu.RUnlock()

	perms, exists := rolePermissions[role]
	if !exists {
		return []string{}
	}
	res := make([]string, 0, len(perms))
	for p := range perms {
		res = append(res, p)
	}
	return res
}

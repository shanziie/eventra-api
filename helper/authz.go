package helper

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PermissionSet mengelola daftar role dan relasi permission secara in-memory.
type PermissionSet struct {
	mu          sync.RWMutex
	permissions map[string]map[string]bool
	roles       map[string]bool
}

var globalPermissions = NewPermissionSet()

// NewPermissionSet membuat instance baru PermissionSet.
func NewPermissionSet() *PermissionSet {
	return &PermissionSet{
		permissions: make(map[string]map[string]bool),
		roles:       make(map[string]bool),
	}
}

// Init memuat mapping role -> permissions ke singleton global.
func Init(data map[string][]string) {
	globalPermissions.Load(data)
}

// Load menyalin mapping ke dalam memory map yang terproteksi mutex.
func (ps *PermissionSet) Load(data map[string][]string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.permissions = make(map[string]map[string]bool)
	ps.roles = make(map[string]bool)

	for role, perms := range data {
		ps.roles[role] = true
		if ps.permissions[role] == nil {
			ps.permissions[role] = make(map[string]bool)
		}
		for _, p := range perms {
			ps.permissions[role][p] = true
		}
	}
}

// Can memeriksa apakah suatu role memiliki permission (fail closed: false bila role/perm tidak ada atau ps nil).
func (ps *PermissionSet) Can(role, permission string) bool {
	if ps == nil {
		return false
	}
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	perms, exists := ps.permissions[role]
	if !exists {
		return false
	}
	return perms[permission]
}

// PermissionsOf mengembalikan daftar permission untuk suatu role (nil-aman).
func (ps *PermissionSet) PermissionsOf(role string) []string {
	if ps == nil {
		return []string{}
	}
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	perms, exists := ps.permissions[role]
	if !exists {
		return []string{}
	}
	res := make([]string, 0, len(perms))
	for p := range perms {
		res = append(res, p)
	}
	return res
}

// KnownRoles mengembalikan seluruh nama role yang terdaftar.
func (ps *PermissionSet) KnownRoles() []string {
	if ps == nil {
		return []string{}
	}
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	res := make([]string, 0, len(ps.roles))
	for r := range ps.roles {
		res = append(res, r)
	}
	return res
}

// IsKnownRole memeriksa apakah role dikenali oleh sistem.
func (ps *PermissionSet) IsKnownRole(role string) bool {
	if ps == nil {
		return false
	}
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.roles[role]
}

// Package-level helpers yang mendelegasikan ke globalPermissions:

func Can(role, permission string) bool {
	return globalPermissions.Can(role, permission)
}

func PermissionsOf(role string) []string {
	return globalPermissions.PermissionsOf(role)
}

func KnownRoles() []string {
	return globalPermissions.KnownRoles()
}

func IsKnownRole(role string) bool {
	return globalPermissions.IsKnownRole(role)
}

// GetPermissionsByRole dipertahankan untuk kompatibilitas service
func GetPermissionsByRole(role string) []string {
	return PermissionsOf(role)
}

// LoadPermissions memuat seluruh relasi role_permissions dari database saat startup (fail closed).
func LoadPermissions(ctx context.Context, db *pgxpool.Pool) error {
	rows, err := db.Query(ctx, "SELECT role_name, permission_name FROM role_permissions")
	if err != nil {
		return err
	}
	defer rows.Close()

	data := make(map[string][]string)
	for rows.Next() {
		var role, perm string
		if err := rows.Scan(&role, &perm); err != nil {
			return err
		}
		data[role] = append(data[role], perm)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	globalPermissions.Load(data)
	return nil
}

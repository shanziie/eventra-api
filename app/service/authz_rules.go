package service

import (
	"eventra-api/helper"
)

// CanAccessUser menentukan apakah seorang pengguna berhak mengakses atau mengubah data user lain.
// Pengguna selalu berhak mengakses datanya sendiri; untuk data orang lain memerlukan permission :any.
func CanAccessUser(currentUserID int, currentRole string, targetUserID int, anyPermission string) bool {
	if currentUserID == targetUserID {
		return true
	}
	return helper.Can(currentRole, anyPermission)
}

// ValidateAssignRole memvalidasi perubahan role pengguna sesuai aturan bisnis BR-U4.
// Pengguna tidak boleh mengubah role dirinya sendiri, dan role baru harus terdaftar di sistem.
func ValidateAssignRole(currentUserID, targetUserID int, newRole string) error {
	if currentUserID == targetUserID {
		return helper.Validation("validasi gagal", map[string]string{"role": "tidak boleh mengubah role diri sendiri"})
	}

	if !helper.IsKnownRole(newRole) {
		return helper.Validation("validasi gagal", map[string]string{"role": "role tidak valid"})
	}

	return nil
}

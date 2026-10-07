package middleware

import (
	"eventra-api/helper"

	"github.com/gofiber/fiber/v2"
)

// RequirePermission memeriksa apakah role pengguna memiliki permission yang dibutuhkan.
// Sesuai AGENTS.md §6, RequireAuth harus selalu dipasang sebelum RequirePermission.
func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		if role == "" || !helper.Can(role, permission) {
			return helper.Forbidden("anda tidak memiliki izin untuk melakukan tindakan ini")
		}
		return c.Next()
	}
}

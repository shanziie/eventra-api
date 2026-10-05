package helper

import (
	"github.com/gofiber/fiber/v2"
)

// CurrentUserID mengambil ID user yang terautentikasi dari Locals context.
func CurrentUserID(c *fiber.Ctx) (int, error) {
	val := c.Locals("user_id")
	id, ok := val.(int)
	if !ok || id <= 0 {
		return 0, Unauthorized("pengguna belum terautentikasi")
	}
	return id, nil
}

// CurrentUserRole mengambil role user yang terautentikasi dari Locals context.
func CurrentUserRole(c *fiber.Ctx) (string, error) {
	val := c.Locals("role")
	role, ok := val.(string)
	if !ok || role == "" {
		return "", Unauthorized("pengguna belum terautentikasi")
	}
	return role, nil
}

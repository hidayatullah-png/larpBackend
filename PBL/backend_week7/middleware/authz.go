package middleware

import (
	"github.com/gofiber/fiber/v2"
	"latihan-fiber/helper"
)

// RequirePermission menolak request yang role-nya tidak memiliki permission tertentu.
// Dipasang pada route yang haknya dapat diputuskan TANPA melihat isi data.
func RequirePermission(perms *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Ambil data user yang sedang login dari context (disisipkan oleh RequireAuth)
		user, ok := helper.CurrentUser(c)
		if !ok {
			// Jika tidak ada, berarti token tidak valid atau RequireAuth lupa dipasang
			return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
		}

		// 2. Periksa apakah role user ini punya permission yang diminta
		if !perms.Can(user.Role, permission) {
			// Jika tidak punya, tolak dengan status 403 Forbidden
			return helper.Fail(c, fiber.StatusForbidden,
				"role "+user.Role+" tidak memiliki hak "+permission)
		}

		// 3. Jika aman, persilakan masuk ke fungsi selanjutnya (Service)
		return c.Next()
	}
}
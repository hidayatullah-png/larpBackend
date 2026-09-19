package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/app/service"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
)

// Dependencies membungkus semua kebutuhan injeksi untuk router,
// menghindari parameter fungsi yang terlalu panjang.
type Dependencies struct {
	Pool        *pgxpool.Pool
	JWT         *helper.JWTManager
	UserService *service.UserService 
	AuthService *service.AuthService
}

// Register memasang seluruh rute API.
func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// --- Publik ---
	api.Get("/health", healthCheck(deps.Pool))

	// --- Autentikasi ---
	// RequireJSON dipasang di level grup auth.
	// Aman karena RequireJSON yang kamu buat sebelumnya otomatis mengabaikan method GET.
	auth := api.Group("/auth", middleware.RequireJSON)

	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)

	// Dilindungi oleh satpam RequireAuth
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	// --- Wajib membawa access token ---
	users := api.Group("/users",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)

	users.Get("/", deps.UserService.List)
	users.Get("/:id", deps.UserService.Get)
	users.Post("/", deps.UserService.Create)
	users.Put("/:id", deps.UserService.Replace)
	users.Patch("/:id", deps.UserService.Patch)
	users.Delete("/:id", deps.UserService.Delete)
}

// healthCheck dipisah menjadi fungsi closure agar fungsi Register lebih bersih.
func healthCheck(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "koneksi database terputus")
		}

		return helper.Success(c, fiber.StatusOK, "layanan dan database berjalan normal", map[string]interface{}{
			"status":   "UP",
			"database": "CONNECTED",
		})
	}
}

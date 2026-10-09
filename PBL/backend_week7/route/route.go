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

type Dependencies struct {
	Pool           *pgxpool.Pool
	JWT            *helper.JWTManager
	Permissions    *helper.PermissionSet
	UserService    *service.UserService
	AuthService    *service.AuthService
	StudentService *service.StudentService
}

// Register memasang seluruh rute API.
func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// --- Publik ---
	api.Get("/health", healthCheck(deps.Pool))

	// --- Autentikasi ---
	auth := api.Group("/auth")
	auth.Post("/register", middleware.RequireJSON, deps.AuthService.Register)
	auth.Post("/login", middleware.RequireJSON, middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", middleware.RequireJSON, deps.AuthService.Refresh)
	auth.Post("/logout", middleware.RequireJSON, deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	perms := deps.Permissions

	// --- RUTE USERS ---
	// RequireJSON dilepas dari level grup agar tidak memblokir GET List (Unduh CSV)
	users := api.Group("/users", middleware.RequireAuth(deps.JWT))

	users.Get("/", middleware.RequirePermission(perms, "user:list"), deps.UserService.List)
	users.Get("/:id", deps.UserService.Get)
	users.Delete("/:id", middleware.RequirePermission(perms, "user:delete"), deps.UserService.Delete)
	
	// RequireJSON dipasang secara spesifik di rute yang membutuhkan body JSON
	users.Post("/", middleware.RequireJSON, middleware.RequirePermission(perms, "user:update:any"), deps.UserService.Create)
	users.Put("/:id", middleware.RequireJSON, deps.UserService.Replace)
	users.Patch("/:id", middleware.RequireJSON, deps.UserService.Patch)

	// --- RUTE STUDENTS ---
	// RequireJSON dilepas dari level grup agar tidak memblokir GET List (Unduh CSV)
	students := api.Group("/students", middleware.RequireAuth(deps.JWT))

	students.Get("/", middleware.RequirePermission(perms, "student:list"), deps.StudentService.List)
	students.Get("/:id", deps.StudentService.Get)
	students.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), deps.StudentService.Delete)

	// RequireJSON dipasang secara spesifik di rute yang membutuhkan body JSON
	students.Post("/", middleware.RequireJSON, middleware.RequirePermission(perms, "student:create"), deps.StudentService.Create)
	students.Put("/:id", middleware.RequireJSON, deps.StudentService.Replace)
	students.Patch("/:id", middleware.RequireJSON, deps.StudentService.Patch)
}

// healthCheck dipisah menjadi fungsi closure agar fungsi Register lebih bersih.
func healthCheck(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			// Mengembalikan AppError alih-alih menggunakan helper.Fail yang sudah dihapus
			return &helper.AppError{
				Status:  fiber.StatusServiceUnavailable,
				Code:    "SERVICE_UNAVAILABLE",
				Message: "koneksi database terputus",
			}
		}

		return helper.Success(c, fiber.StatusOK, "layanan dan database berjalan normal", map[string]interface{}{
			"status":   "UP",
			"database": "CONNECTED",
		})
	}
}
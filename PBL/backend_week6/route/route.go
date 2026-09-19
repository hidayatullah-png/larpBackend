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

// Dependencies membungkus semua kebutuhan injeksi untuk router, menghindari parameter fungsi yang terlalu panjang.
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
	auth := api.Group("/auth", middleware.RequireJSON)

	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)

	// Dilindungi oleh satpam RequireAuth
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	// Variabel perms agar lebih singkat saat dipanggil di middleware
	perms := deps.Permissions

	// --- RUTE USERS ---
	users := api.Group("/users",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)

	// Rute yang dicegat Middleware (tidak butuh lihat data)
	users.Get("/", middleware.RequirePermission(perms, "user:list"), deps.UserService.List)
	users.Post("/", middleware.RequirePermission(perms, "user:update:any"), deps.UserService.Create)
	users.Delete("/:id", middleware.RequirePermission(perms, "user:delete"), deps.UserService.Delete)
	
	// Jika ada fitur Assign Role
	// users.Patch("/:id/role", middleware.RequirePermission(perms, "role:assign"), deps.UserService.AssignRole)

	// Rute yang diloloskan dari Middleware (hak diurus oleh Service)
	users.Get("/:id", deps.UserService.Get)
	users.Put("/:id", deps.UserService.Replace)
	users.Patch("/:id", deps.UserService.Patch)

	// --- RUTE STUDENTS 
	students := api.Group("/students",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)

	// Rute yang dicegat Middleware 
	students.Get("/", middleware.RequirePermission(perms, "student:list"), deps.StudentService.List)
	students.Post("/", middleware.RequirePermission(perms, "student:create"), deps.StudentService.Create)
	students.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), deps.StudentService.Delete)

	// Rute yang diloloskan dari Middleware agar owner_id dicek di Service (C.2 Poin 3)
	students.Get("/:id", deps.StudentService.Get)
	students.Put("/:id", deps.StudentService.Replace)
	students.Patch("/:id", deps.StudentService.Patch)
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
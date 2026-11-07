package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"latihan-fiber/app/service"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
)

type Dependencies struct {
	Pool                *pgxpool.Pool
	Mongo               *mongo.Client
	JWT                 *helper.JWTManager
	Permissions         *helper.PermissionSet
	UserService         *service.UserService
	AuthService         *service.AuthService
	StudentService      *service.StudentService
	CourseService       *service.CourseService
	SupervisorService   *service.SupervisorService
	ReviewService       *service.ReviewService
	PortfolioService    *service.PortfolioService
	NotificationService *service.NotificationService
}

// Register memasang seluruh rute API.
func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// --- Publik ---
	api.Get("/health", healthCheck(deps.Pool, deps.Mongo))

	// --- Autentikasi ---
	auth := api.Group("/auth")
	auth.Post("/register", middleware.RequireJSON, deps.AuthService.Register)
	auth.Post("/login", middleware.RequireJSON, middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", middleware.RequireJSON, deps.AuthService.Refresh)
	auth.Post("/logout", middleware.RequireJSON, deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	perms := deps.Permissions

	// --- RUTE USERS ---
	users := api.Group("/users", middleware.RequireAuth(deps.JWT))
	users.Get("/", middleware.RequirePermission(perms, "user:list"), deps.UserService.List)
	users.Get("/:id", deps.UserService.Get)
	users.Delete("/:id", middleware.RequirePermission(perms, "user:delete"), deps.UserService.Delete)
	users.Post("/", middleware.RequireJSON, middleware.RequirePermission(perms, "user:update:any"), deps.UserService.Create)
	users.Put("/:id", middleware.RequireJSON, deps.UserService.Replace)
	users.Patch("/:id", middleware.RequireJSON, deps.UserService.Patch)

	// --- RUTE STUDENTS ---
	students := api.Group("/students", middleware.RequireAuth(deps.JWT))
	students.Get("/", middleware.RequirePermission(perms, "student:list"), deps.StudentService.List)
	students.Get("/:id", deps.StudentService.Get)
	students.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), deps.StudentService.Delete)
	students.Post("/", middleware.RequireJSON, middleware.RequirePermission(perms, "student:create"), deps.StudentService.Create)
	students.Put("/:id", middleware.RequireJSON, deps.StudentService.Replace)
	students.Patch("/:id", middleware.RequireJSON, deps.StudentService.Patch)

	// kelas dan KRS
	courses := api.Group("/courses",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))

	courses.Get("/", deps.CourseService.List)
	courses.Get("/mine", deps.CourseService.Mine)
	courses.Get("/:id", deps.CourseService.Get)
	courses.Post("/:id/enroll", deps.CourseService.Enroll)
	courses.Delete("/:id/enroll", deps.CourseService.Drop)
	courses.Patch("/:id", middleware.RequirePermission(deps.Permissions, "course:manage"),
		deps.CourseService.UpdateQuota)

	courses.Get("/:id/reviews", deps.ReviewService.ListByCourse)
	courses.Get("/:id/reviews/summary", deps.ReviewService.Summary)
	courses.Post("/:id/reviews", deps.ReviewService.Create)
	// Nanti di Langkah 8, aktifkan baris-baris ini:
	// courses.Get("/:id/reviews", deps.ReviewService.ListByCourse)
	// courses.Get("/:id/reviews/summary", deps.ReviewService.Summary)
	// courses.Post("/:id/reviews", deps.ReviewService.Create)

	enrollments := api.Group("/enrollments",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	enrollments.Post("/swap", deps.CourseService.Swap)

	// Dosen Pembimbing Tugas Akhir (Tugas Mandiri D.2)
	supervisors := api.Group("/supervisors",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	supervisors.Get("/", deps.SupervisorService.List)
	supervisors.Post("/:id/assign", deps.SupervisorService.Assign)
	supervisors.Patch("/:id", middleware.RequirePermission(deps.Permissions, "supervision:manage"),
		deps.SupervisorService.UpdateQuota)

	supervisions := api.Group("/supervisions",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	supervisions.Post("/switch", deps.SupervisorService.Switch)

	//--review--
	reviews := api.Group("/reviews",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	reviews.Get("/:id", deps.ReviewService.Get)
	reviews.Patch("/:id", deps.ReviewService.Update)
	reviews.Delete("/:id", deps.ReviewService.Delete)
	reviews.Post("/:id/replies", deps.ReviewService.Reply)
	reviews.Post("/:id/helpful", deps.ReviewService.MarkHelpful)

	// --- Portofolio Mahasiswa (Tugas Mandiri C.2) ---
	students.Get("/:id/portfolio", deps.PortfolioService.Get)
	students.Put("/:id/portfolio", middleware.RequireJSON, deps.PortfolioService.Upsert)
	students.Post("/:id/portfolio/projects", middleware.RequireJSON, deps.PortfolioService.AddProject)
	students.Delete("/:id/portfolio/projects/:projectId", deps.PortfolioService.RemoveProject)
	portfolios := api.Group("/portfolios", middleware.RequireAuth(deps.JWT))
	portfolios.Get("/stats/technologies", deps.PortfolioService.TopTechnologies)

	// --- notifikasi (diisi oleh worker, dibaca lewat API)
	notifications := api.Group("/notifications",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	notifications.Get("/mine", deps.NotificationService.Mine)

	admin := api.Group("/admin",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT))
	admin.Get("/queue", middleware.RequirePermission(deps.Permissions, "queue:monitor"), deps.NotificationService.Monitor)
}

// healthCheck kini memeriksa DUA database sesuai
func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("PostgreSQL tidak dapat dihubungi")
		}
		if err := mongoClient.Ping(ctx, nil); err != nil {
			return helper.ServiceUnavailable("MongoDB tidak dapat dihubungi")
		}

		return helper.Success(c, fiber.StatusOK, "server, PostgreSQL, dan MongoDB berjalan", nil)
	}
}

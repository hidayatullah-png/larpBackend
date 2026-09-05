package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/app/repository"
	"latihan-fiber/middleware"
	"latihan-fiber/app/service"
	"latihan-fiber/helper"
)

func RegisterRoute(app *fiber.App, repo repository.StudentRepository, db*pgxpool.Pool) {
	// inisialisasi service
	studentService := service.NewStudentService(repo)

	//grup route utama
	api := app.Group("/api/v1")

	//endpoint untuk health check
	api.Get("/health", func(c*fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil{
				return helper.Fail(c, fiber.StatusServiceUnavailable, "koneksi database terputus")
		}

		return helper.Success(c, fiber.StatusOK, "layanan dan database berjalan normal", map[string]interface{}{
			"status": "UP",
			"database": "CONNECTED",
		})
	})
	//rute yang TIDAK butuh RequireJSON (GET, DELETE)
	api.Get("/students", studentService.List)
	api.Get("/students/:id", studentService.Get)
	api.Delete("/students/:id", studentService.Delete)

	//sub-grup khusus rute yang MEMBUTUHKAN body JSON (POST, PUT, PATCH)
	mutationGroup := api.Group("")
	mutationGroup.Use(middleware.RequireJSON)

	mutationGroup.Post("", studentService.Create)
	mutationGroup.Put("/:id", studentService.Replace)
	mutationGroup.Patch("/:id", studentService.Patch)
}
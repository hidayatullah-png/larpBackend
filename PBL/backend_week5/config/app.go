package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"latihan-fiber/app/service"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
	"latihan-fiber/route"
)

// NewApp merakit aplikasi: membuat instance Fiber, menambahkan middleware, dan mendaftarkan route.
// Fungsi ini menerima logger, pool database, dan service sebagai parameter agar bisa diteruskan ke middleware dan route.
func NewApp(logger *slog.Logger, pool *pgxpool.Pool, studentService service.StudentService) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "latihan-fiber"),
		ErrorHandler: newErrorHandler(logger),
	})

	// Perbaikan: gunakan huruf besar R
	middleware.Register(app, logger)

	// Perbaikan: gunakan studentService sesuai dengan parameter
	route.RegisterRoute(app, pool, studentService) // (Sesuaikan nama fungsi Register/RegisterRoute sesuai yang ada di file route.go milikmu)

	// Penampung terakhir untuk URL yang tidak dikenal
	app.Use(func(c *fiber.Ctx) error {
		// Perbaikan: StatusNotFound
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}

// newErrorHandler adalah jaring pengaman terakhir: error yang tidak tertangani di service berakhir di sini dengan format yang tetap konsisten.
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		message := "terjadi error pada server"

		if e, ok := err.(*fiber.Error); ok {
			status = e.Code
			message = e.Message
		}

		logger.Error("unhandled_error",
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.String("error", err.Error()),
		)

		return helper.Fail(c, status, message)
	}
}

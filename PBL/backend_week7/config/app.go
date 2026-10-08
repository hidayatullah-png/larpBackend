package config

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/helper"
	"latihan-fiber/middleware"
	"latihan-fiber/route"
)

// NewApp merakit aplikasi: membuat instance Fiber, memasang middleware,
// lalu mendaftarkan route menggunakan Dependencies yang dikirim dari main.go.
func NewApp(logger *slog.Logger, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "Praktikum Backend Lanjut"),
		ErrorHandler: newErrorHandler(logger),
		// Membatasi ukuran body mencegah satu request besar menghabiskan
		// memori server (denial of service yang paling murah dilakukan).
		BodyLimit: 1 * 1024 * 1024, // 1 MB

	})

	// 1. Pasang Middleware Umum
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	middleware.Register(app, logger, allowedOrigins)

	// 2. Daftarkan seluruh rute API
	route.Register(app, deps)

	// 3. Penampung terakhir untuk URL yang tidak dikenal
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}

// newErrorHandler adalah jaring pengaman terakhir untuk error yang tidak tertangani.
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

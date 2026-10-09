package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/helper"
	"latihan-fiber/middleware"
	"latihan-fiber/route"
	"latihan-fiber/app/model"
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
		return helper.NotFound("endpoint tidak ditemukan")
	})

	return app
}

// newErrorHandler adalah SATU-SATUNYA tempat error berubah menjadi
// response HTTP di seluruh aplikasi.
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		requestID := helper.RequestID(c)
		var appErr *helper.AppError

		switch {
		case errors.As(err, &appErr):
			// Kegagalan yang sudah kita rencanakan
		case errors.Is(err, fiber.ErrRequestEntityTooLarge):
			appErr = &helper.AppError{
				Status:  fiber.StatusRequestEntityTooLarge,
				Code:    "PAYLOAD_TOO_LARGE",
				Message: "ukuran body melebihi batas yang diizinkan",
			}
		default:
			// Kegagalan yang tidak kita duga
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				appErr = &helper.AppError{
					Status:  fiberErr.Code,
					Code:    "HTTP_ERROR",
					Message: fiberErr.Message,
				}
			} else {
				appErr = helper.Internal(err)
			}
		}

		// PERBAIKAN BUG 3: Kondisi diubah menjadi >= 500 (StatusInternalServerError). 5xx harus dicatat sebagai ERROR, sedangkan 4xx dicatat sebagai WARN
		if appErr.Status >= fiber.StatusInternalServerError {

			// PERBAIKAN BUG 4: Memastikan cause tidak bernilai nil sebelum memanggil .Error() untuk mencegah server mengalami panic saat terjadi error 4xx biasa
			cause := appErr.Cause()
			causeText := ""
			if cause != nil {
				causeText = cause.Error()
			} else {
				causeText = appErr.Message
			}

			logger.Error("request_failed",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status),
				slog.String("error", causeText))
		} else {
			logger.Warn("request_rejected",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status))
		}

		return c.Status(appErr.Status).JSON(model.ErrorResponse{
			Success:   false,
			Code:      appErr.Code,
			Message:   appErr.Message,
			Fields:    appErr.Fields,
			RequestID: requestID,
		})
	}
}

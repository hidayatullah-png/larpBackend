package middleware

import (
	"log/slog"
	"strings"
	"time"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"latihan-fiber/helper"
)

// Register memasang seluruh middleware umum yang dibutuhkan ke dalam *fiber.App
func Register(app *fiber.App, logger *slog.Logger, allowedOrigins string) {
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(corsPolicy(allowedOrigins)) // BERUBAH: membatasi domain yang diizinkan
	app.Use(RequestLogger(logger))
}

// corsPolicy membatasi origin yang boleh memanggil API.
// cors.New() tanpa konfigurasi mengizinkan SEMUA origin
func corsPolicy(allowedOrigins string) fiber.Handler {
	if strings.TrimSpace(allowedOrigins) == "" {
		allowedOrigins = "http://localhost:5173"
	}

	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	})
}

// RequestLogger mencatat setiap permintaan HTTP yang masuk, termasuk metode, jalur, status, durasi,
// dan identitas pengguna.
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		
		// 1. Lanjutkan eksekusi ke handler berikutnya
		err := c.Next()
		
		requestID, _ := c.Locals("requestid").(string)
		
		// 2. Ambil status bawaan
		status := c.Response().StatusCode()
		
		// 3. KOREKSI STATUS LOG
		// Jika terjadi error, kita tidak bisa mengandalkan c.Response().StatusCode() karena ErrorHandler belum merakit balasan HTTP-nya.
		if err != nil {
			var appErr *helper.AppError
			if errors.As(err, &appErr) {
				status = appErr.Status // Ambil status asli dari AppError (misal 404, 422)
			} else {
				status = fiber.StatusInternalServerError // Error tak terduga selalu 500
			}
		}

		// 4. Catat ke log dengan variabel `status` yang sudah dikoreksi
		logger.Info("http_request",
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status), // <-- PASTIKAN MENGGUNAKAN VARIABEL `status`
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		)

		// 5. Kembalikan error agar ErrorHandler terpusat bisa memprosesnya
		return err
	}
}

var methodWithBody = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

// RequireJSON menolak request berisi body yang Content-Type bukan application/json, kecuali untuk metode GET, HEAD, dan DELETE.
func RequireJSON(c *fiber.Ctx) error {
	if methodWithBody[c.Method()] {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return helper.Fail(c, fiber.StatusUnsupportedMediaType, "Content-Type must be application/json")
		}
	}
	return c.Next()
}


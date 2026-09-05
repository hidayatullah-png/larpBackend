package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"latihan-fiber/helper"
)

// Register memasang seluruh middleware yang dibutuhkan ke dalam *fiber.App
func Register(app *fiber.App, logger *slog.Logger) {
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))
	app.Use(RequestLogger(logger))
}

// RequestLogger mencatat setiap permintaan HTTP yang masuk, termasuk metode, jalur, status, dan durasi.
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next() // memanggil handler berikutnya
		requestID, _ := c.Locals("requestid").(string)

		logger.Info("http_request",
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		)
		return err
	}
}

var methodWithBody = map[string]bool{
	fiber.MethodPost: true,
	fiber.MethodPut: true,
	fiber.MethodPatch: true,
}

//RequireJSON menolak request berisi body yang Content-Type bukan application/json, kecuali untuk metode GET, HEAD, dan DELETE.
func RequireJSON(c *fiber.Ctx) error{
	if methodWithBody[c.Method()]{
		ct := c.Get ("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON){
			return helper.Fail(c, fiber.StatusUnsupportedMediaType,
				"Content-Type must be application/json")
			
		}
	}
	return c.Next()
}
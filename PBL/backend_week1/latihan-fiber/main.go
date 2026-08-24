package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error { //route
		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":3000")) //log fatal buat menghentikan dan mencetak errors 
}
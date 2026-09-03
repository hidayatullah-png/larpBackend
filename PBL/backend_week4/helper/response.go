package helper

import (
	
	"github.com/gofiber/fiber/v2"
	"latihan-fiber/app/model"
)

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func SuccessList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location) // memberi tahu klien di mana sumber daya baru berada
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent) // 204: berhasil, tanpa body
}

func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{Success: false, Message: message})
}

func FailValidation(c *fiber.Ctx, errs map[string]string) error {
	// Menaruh pesan error validasi ke dalam Data karena struct WebResponse tidak punya field Errors
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Success: false, Message: "Validasi gagal", Data: errs,
	})
}
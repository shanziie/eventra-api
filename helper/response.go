package helper

import (
	"eventra-api/app/model"

	"github.com/gofiber/fiber/v2"
)

// Success mengirim response 200 dengan data tunggal.
func Success(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessList mengirim response 200 dengan data dan meta offset pagination.
func SuccessList(c *fiber.Ctx, message string, data any, meta model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebListResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// SuccessCursor mengirim response 200 dengan data dan meta cursor pagination.
func SuccessCursor(c *fiber.Ctx, message string, data any, meta model.CursorMeta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebCursorResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created mengirim response 201 dengan header Location.
func Created(c *fiber.Ctx, location, message string, data any) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// NoContent mengirim response 204 tanpa body.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

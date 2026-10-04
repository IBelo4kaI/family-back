package response

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
)

func JSON(c fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(data)
}

func Error(c fiber.Ctx, status int, err error) error {
	return JSON(c, status, map[string]string{"error": err.Error()})
}

func ServerError(c fiber.Ctx) error {
	return JSON(c, http.StatusInternalServerError, map[string]string{"error": "На сервере произошла ошибка, попробуйте позже"})
}

func BadRequest(c fiber.Ctx) error {
	return JSON(c, http.StatusBadRequest, map[string]string{"error": "Проверьте параметры запроса и повторите попытку"})
}

func Success(c fiber.Ctx, data any) error {
	return JSON(c, http.StatusOK, data)
}

func Created(c fiber.Ctx, data any) error {
	return JSON(c, http.StatusCreated, data)
}

func Deleted(c fiber.Ctx) error {
	return c.SendStatus(http.StatusNoContent)
}

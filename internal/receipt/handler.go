package receipt

import (
	"errors"
	"net/http"

	"family/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func actor(c fiber.Ctx) Actor {
	userID, _ := c.Locals("user_id").(uuid.UUID)
	familyID, _ := c.Locals("family_id").(uuid.UUID)
	return Actor{UserID: userID, FamilyID: familyID}
}

func (h Handler) Check(c fiber.Ctx) error {
	var in CheckRequest
	if err := c.Bind().Body(&in); err != nil {
		return response.BadRequest(c)
	}
	r, err := h.service.Check(c.RequestCtx(), in.QRRaw)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, r)
}

func (h Handler) Save(c fiber.Ctx) error {
	var in SaveInput
	if err := c.Bind().Body(&in); err != nil {
		return response.BadRequest(c)
	}
	t, err := h.service.Save(c.RequestCtx(), actor(c), in)
	if err != nil {
		return mapError(c, err)
	}
	return response.Created(c, t)
}

func (h Handler) Items(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c)
	}
	items, err := h.service.Items(c.RequestCtx(), actor(c), id)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, items)
}

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return response.Error(c, http.StatusBadRequest, err)
	case errors.Is(err, ErrNotFound):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrDuplicate):
		return response.Error(c, http.StatusConflict, err)
	case errors.Is(err, ErrCheckFailed), errors.Is(err, ErrRefund):
		return response.Error(c, http.StatusUnprocessableEntity, err)
	case errors.Is(err, ErrNoToken):
		return response.Error(c, http.StatusServiceUnavailable, err)
	default:
		return response.ServerError(c)
	}
}

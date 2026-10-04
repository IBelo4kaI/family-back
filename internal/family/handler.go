package family

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

// ids достаёт user_id и family_id из Locals; ok=false — пользователь без семьи.
func ids(c fiber.Ctx) (userID, familyID uuid.UUID, ok bool) {
	userID, _ = c.Locals("user_id").(uuid.UUID)
	familyID, ok = c.Locals("family_id").(uuid.UUID)
	return
}

func (h Handler) Get(c fiber.Ctx) error {
	_, familyID, ok := ids(c)
	if !ok {
		return response.Error(c, http.StatusNotFound, ErrNoFamily)
	}
	f, err := h.service.Get(c.RequestCtx(), familyID)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, f)
}

func (h Handler) CreateInvite(c fiber.Ctx) error {
	userID, familyID, ok := ids(c)
	if !ok {
		return response.Error(c, http.StatusNotFound, ErrNoFamily)
	}
	inv, err := h.service.CreateInvite(c.RequestCtx(), familyID, userID)
	if err != nil {
		return mapError(c, err)
	}
	return response.Created(c, inv)
}

func (h Handler) Kick(c fiber.Ctx) error {
	userID, familyID, ok := ids(c)
	if !ok {
		return response.Error(c, http.StatusNotFound, ErrNoFamily)
	}
	memberID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c)
	}
	if err := h.service.Kick(c.RequestCtx(), familyID, userID, memberID); err != nil {
		return mapError(c, err)
	}
	return response.Deleted(c)
}

func (h Handler) Leave(c fiber.Ctx) error {
	userID, familyID, ok := ids(c)
	if !ok {
		return response.Error(c, http.StatusNotFound, ErrNoFamily)
	}
	if err := h.service.Leave(c.RequestCtx(), familyID, userID); err != nil {
		return mapError(c, err)
	}
	return response.Deleted(c)
}

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotOwner):
		return response.Error(c, http.StatusForbidden, err)
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrNoFamily):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrCannotKick):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.ServerError(c)
	}
}

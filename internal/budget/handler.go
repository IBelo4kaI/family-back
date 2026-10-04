package budget

import (
	"context"
	"errors"
	"net/http"

	"family/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var errNoFamily = errors.New("вы не состоите в семье")

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// RequireFamily пускает дальше только пользователей, состоящих в семье.
func RequireFamily(c fiber.Ctx) error {
	if _, ok := c.Locals("family_id").(uuid.UUID); !ok {
		return response.Error(c, http.StatusForbidden, errNoFamily)
	}
	return c.Next()
}

func actor(c fiber.Ctx) Actor {
	userID, _ := c.Locals("user_id").(uuid.UUID)
	familyID, _ := c.Locals("family_id").(uuid.UUID)
	return Actor{UserID: userID, FamilyID: familyID}
}

// create/update/delete сводят повторяющийся разбор тела и id к одному месту.
func create[In, Out any](h Handler, fn func(context.Context, Actor, In) (Out, error)) fiber.Handler {
	return func(c fiber.Ctx) error {
		var in In
		if err := c.Bind().Body(&in); err != nil {
			return response.BadRequest(c)
		}
		out, err := fn(c.RequestCtx(), actor(c), in)
		if err != nil {
			return mapError(c, err)
		}
		return response.Created(c, out)
	}
}

func update[In, Out any](h Handler, fn func(context.Context, Actor, uuid.UUID, In) (Out, error)) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return response.BadRequest(c)
		}
		var in In
		if err := c.Bind().Body(&in); err != nil {
			return response.BadRequest(c)
		}
		out, err := fn(c.RequestCtx(), actor(c), id, in)
		if err != nil {
			return mapError(c, err)
		}
		return response.Success(c, out)
	}
}

func remove(fn func(context.Context, Actor, uuid.UUID) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return response.BadRequest(c)
		}
		if err := fn(c.RequestCtx(), actor(c), id); err != nil {
			return mapError(c, err)
		}
		return response.Deleted(c)
	}
}

func (h Handler) Load(c fiber.Ctx) error {
	data, err := h.service.Load(c.RequestCtx(), actor(c))
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, data)
}

func (h Handler) TopUpGoal(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c)
	}
	var in TopUpRequest
	if err := c.Bind().Body(&in); err != nil {
		return response.BadRequest(c)
	}
	g, err := h.service.TopUpGoal(c.RequestCtx(), actor(c), id, in.Amount)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, g)
}

func settle(fn func(context.Context, Actor, uuid.UUID, string) (SettleResult, error)) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return response.BadRequest(c)
		}
		var in SettleRequest
		if err := c.Bind().Body(&in); err != nil {
			return response.BadRequest(c)
		}
		res, err := fn(c.RequestCtx(), actor(c), id, in.Action)
		if err != nil {
			return mapError(c, err)
		}
		return response.Success(c, res)
	}
}

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return response.Error(c, http.StatusBadRequest, err)
	case errors.Is(err, ErrNotFound):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrClosed), errors.Is(err, ErrLimitExists):
		return response.Error(c, http.StatusConflict, err)
	default:
		return response.ServerError(c)
	}
}

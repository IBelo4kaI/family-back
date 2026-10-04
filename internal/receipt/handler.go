package receipt

import (
	"errors"
	"io"
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

const maxImageSize = 8 << 20

var imageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

func (h Handler) CheckImage(c fiber.Ctx) error {
	header, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c)
	}
	if header.Size > maxImageSize {
		return response.Error(c, http.StatusRequestEntityTooLarge, ErrTooLarge)
	}

	f, err := header.Open()
	if err != nil {
		return response.BadRequest(c)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxImageSize+1))
	if err != nil || len(data) > maxImageSize {
		return response.BadRequest(c)
	}
	if !imageTypes[http.DetectContentType(data)] {
		return response.Error(c, http.StatusBadRequest, ErrInvalidInput)
	}

	r, err := h.service.CheckImage(c.RequestCtx(), header.Filename, data)
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
	case errors.Is(err, ErrTooLarge):
		return response.Error(c, http.StatusRequestEntityTooLarge, err)
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

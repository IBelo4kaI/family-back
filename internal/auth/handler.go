package auth

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

func (h Handler) Register(c fiber.Ctx) error {
	var body RegisterRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	s, err := h.service.Register(c.RequestCtx(), body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Created(c, s)
}

func (h Handler) Join(c fiber.Ctx) error {
	var body JoinRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	s, err := h.service.Join(c.RequestCtx(), body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Created(c, s)
}

func (h Handler) Login(c fiber.Ctx) error {
	var body LoginRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	s, err := h.service.Login(c.RequestCtx(), body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, s)
}

func (h Handler) Refresh(c fiber.Ctx) error {
	var body RefreshRequest
	if err := c.Bind().Body(&body); err != nil || body.RefreshToken == "" {
		return response.BadRequest(c)
	}
	t, err := h.service.Refresh(c.RequestCtx(), body.RefreshToken)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, t)
}

func (h Handler) Logout(c fiber.Ctx) error {
	var body RefreshRequest
	if err := c.Bind().Body(&body); err != nil || body.RefreshToken == "" {
		return response.BadRequest(c)
	}
	if err := h.service.Logout(c.RequestCtx(), body.RefreshToken); err != nil {
		return response.ServerError(c)
	}
	return response.Deleted(c)
}

func (h Handler) Me(c fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uuid.UUID)
	p, err := h.service.Me(c.RequestCtx(), userID)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, p)
}

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrWeakPassword), errors.Is(err, ErrInvalidInvite):
		return response.Error(c, http.StatusBadRequest, err)
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidRefresh):
		return response.Error(c, http.StatusUnauthorized, err)
	case errors.Is(err, ErrEmailTaken):
		return response.Error(c, http.StatusConflict, err)
	case errors.Is(err, ErrNotFound):
		return response.Error(c, http.StatusNotFound, err)
	default:
		return response.ServerError(c)
	}
}

package server

import (
	"errors"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
)

func domainErrorStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, types.ErrInvalidArgument):
		return fiber.StatusBadRequest, true
	case errors.Is(err, types.ErrUnauthorized):
		return fiber.StatusUnauthorized, true
	case errors.Is(err, types.ErrForbidden):
		return fiber.StatusForbidden, true
	case errors.Is(err, types.ErrNotFound):
		return fiber.StatusNotFound, true
	case errors.Is(err, types.ErrAlreadyExists), errors.Is(err, types.ErrConflict):
		return fiber.StatusConflict, true
	case errors.Is(err, types.ErrRateLimited):
		return fiber.StatusTooManyRequests, true
	case errors.Is(err, types.ErrUnavailable):
		return fiber.StatusServiceUnavailable, true
	case errors.Is(err, types.ErrTimeout):
		return fiber.StatusGatewayTimeout, true
	case errors.Is(err, types.ErrNotImplemented):
		return fiber.StatusNotImplemented, true
	case errors.Is(err, types.ErrProvider):
		return fiber.StatusBadGateway, true
	case errors.Is(err, types.ErrInternal):
		return fiber.StatusInternalServerError, true
	default:
		return fiber.StatusInternalServerError, false
	}
}

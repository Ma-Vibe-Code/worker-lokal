package err

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/Ma-Vibe-Code/worker-lokal/pkg/model"
)

// ErrorHandler provides centralized error interception and formats standard ResponseError JSON envelopes.
func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}

	response := model.ResponseError[any]{
		ResponseEntity: model.ResponseEntity[any]{
			Code:    code,
			Status:  false,
			Message: message,
			Data:    nil,
		},
		Path: c.Path(),
	}

	return c.Status(code).JSON(response)
}

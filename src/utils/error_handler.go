// utils/error_handler.go
package utils

import (
	"user-auth-profile-service/src/responses"

	"github.com/gofiber/fiber/v2"
)

// Common error codes
const (
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeDuplicate     = "DUPLICATE_ENTRY"
	ErrCodeBadRequest    = "BAD_REQUEST"
)

func RespondWithError(c *fiber.Ctx, status int, code string, message string, err error) error {
	details := make(map[string]string)
	if err != nil {
		details["error"] = err.Error()
	}
	return responses.SendErrorResponse(c, status, code, message, details)
}

func RespondWithValidationError(c *fiber.Ctx, details map[string]string) error {
	return responses.SendErrorResponse(
		c,
		fiber.StatusBadRequest,
		ErrCodeValidation,
		"Validation failed",
		details,
	)
}

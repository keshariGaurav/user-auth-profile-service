package responses

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Common error codes
const (
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeUnauthorized  = "UNAUTHORIZED"
	ErrCodeInternalError = "INTERNAL_ERROR"
	ErrCodeDuplicate     = "DUPLICATE_ENTRY"
	ErrCodeBadRequest    = "BAD_REQUEST"
)

type Response struct {
	Status      int         `json:"status"`
	Success     bool        `json:"success"`
	Message     string      `json:"message"`
	Data        interface{} `json:"data,omitempty"`
	Error       *ErrorInfo  `json:"error,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
	RequestID   string      `json:"requestId"`
}

type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func SendSuccessResponse(c *fiber.Ctx, status int, message string, data interface{}) error {
	response := Response{
		Status:    status,
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC(),
		RequestID: uuid.New().String(),
	}
	return c.Status(status).JSON(response)
}

func SendErrorResponse(c *fiber.Ctx, status int, code string, message string, details map[string]string) error {
	response := Response{
		Status:    status,
		Success:   false,
		Timestamp: time.Now().UTC(),
		RequestID: uuid.New().String(),
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	return c.Status(status).JSON(response)
}

func SendValidationError(c *fiber.Ctx, details map[string]string) error {
	return SendErrorResponse(
		c,
		fiber.StatusBadRequest,
		ErrCodeValidation,
		"Validation failed",
		details,
	)
}
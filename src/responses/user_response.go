package responses

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

func NewResponse(status int, success bool, message string, data interface{}) *Response {
	return &Response{
		Status:    status,
		Success:   success,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC(),
		RequestID: uuid.New().String(),
	}
}

func NewErrorResponse(status int, code string, message string, details map[string]string) *Response {
	return &Response{
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
}

func SendSuccessResponse(c *fiber.Ctx, status int, message string, data interface{}) error {
	return c.Status(status).JSON(NewResponse(status, true, message, data))
}

func SendErrorResponse(c *fiber.Ctx, status int, code string, message string, details map[string]string) error {
	return c.Status(status).JSON(NewErrorResponse(status, code, message, details))
}

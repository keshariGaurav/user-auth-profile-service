package middleware

import (
	"strings"
	"time"

	"user-auth-profile-service/src/utils"

	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{"error": "No authorization header"})
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token format"})
	}

	claims, err := utils.ParseJWT(tokenString)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
	}

	// Check token expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return c.Status(401).JSON(fiber.Map{"error": "Token has expired"})
		}
	}

	// Store email in context for use in protected routes
	if email, ok := claims["email"].(string); ok {
		c.Locals("email", email)
	} else {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	return c.Next()
}

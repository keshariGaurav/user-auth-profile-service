package main

import (
	"log"
	"user-auth-profile-service/src/routes"
	"github.com/gofiber/fiber/v2"
)

func main() {

	app := fiber.New()
	routes.UserRoute(app)
	routes.AuthRoute(app)

	if err := app.Listen(":6400"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

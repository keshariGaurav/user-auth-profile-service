package main

import (
	"log"
	"os"

	"user-auth-profile-service/src/configs"
	"user-auth-profile-service/src/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	if err := os.Setenv("AWS_ACCESS_KEY_ID", os.Getenv("AWS_ACCESS_KEY_ID")); err != nil {
		log.Fatalf("Failed to set AWS_ACCESS_KEY_ID: %v", err)
	}
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", os.Getenv("AWS_SECRET_ACCESS_KEY")); err != nil {
		log.Fatalf("Failed to set AWS_SECRET_ACCESS_KEY: %v", err)
	}

	if err := os.Setenv("AWS_REGION", os.Getenv("AWS_REGION")); err != nil {
		log.Fatalf("Failed to set AWS_REGION: %v", err)
	}
	app := fiber.New()
	configs.ConnectDB()
	routes.UserRoute(app)
	routes.AuthRoute(app)

	if err := app.Listen(":6000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

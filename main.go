package main

import (
	"log"

	"user-auth-profile-service/src/configs"
	"user-auth-profile-service/src/rabbitmq"
	"user-auth-profile-service/src/routes"

	"github.com/gofiber/fiber/v2"
)
var (
	rabbitConn    *rabbitmq.Connection
	emailProducer *rabbitmq.Producer
)
func main() {
	cfg := configs.LoadEnv()

	// Initialize RabbitMQ connection
	var err error
	rabbitConn, err = rabbitmq.NewConnection(cfg.AmqpURL)
	if err != nil {
		log.Fatal("Failed to establish RabbitMQ connection:", err)
	}
	defer rabbitConn.Close()

	// Initialize producer
	emailProducer, err = rabbitmq.NewProducer(rabbitConn.Channel, "email_queue", true)
	if err != nil {
		log.Fatal("Failed to create producer:", err)
	}


	app := fiber.New()
	configs.ConnectDB()
	if err := configs.SetupAllIndexes(); err != nil {
	log.Fatal(err)
	}
	routes.UserRoute(app)
	routes.AuthRoute(app)

	if err := app.Listen(":6400"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

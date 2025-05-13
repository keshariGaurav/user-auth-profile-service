package routes

import (
	"user-auth-profile-service/src/controllers"
	"user-auth-profile-service/src/middleware"

	"github.com/gofiber/fiber/v2"
)

func UserRoute(app *fiber.App) {
	// Protected routes that require authentication
	app.Post("/user", middleware.AuthMiddleware, controllers.CreateUser)
	app.Get("/user/:userId", middleware.AuthMiddleware, controllers.GetAUser)
	app.Put("/user/:userId", middleware.AuthMiddleware, controllers.EditAUser)
	app.Delete("/user/:userId", middleware.AuthMiddleware, controllers.DeleteAUser)
	app.Get("/users", middleware.AuthMiddleware, controllers.GetAllUsers)
}

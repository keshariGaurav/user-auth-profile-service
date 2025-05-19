package routes

import (
	"user-auth-profile-service/src/controllers"
	"user-auth-profile-service/src/middleware"

	"github.com/gofiber/fiber/v2"
)

func AuthRoute(app *fiber.App) {
	app.Post("auth/register", controllers.Register)
	app.Post("auth/login", controllers.Login)
	app.Patch("auth/update-password",middleware.AuthMiddleware, controllers.UpdatePassword)
	app.Post("auth/forgot-password", controllers.ForgotPassword)
	app.Post("auth/reset-password", controllers.ResetPassword)
	app.Post("auth/verify-otp",  controllers.VerifyOTP)
}

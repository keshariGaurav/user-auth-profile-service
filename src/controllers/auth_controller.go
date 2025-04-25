package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"user-auth-profile-service/src/configs"
	"user-auth-profile-service/src/models"
	"user-auth-profile-service/src/structure"
	"user-auth-profile-service/src/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var authCol *mongo.Collection = configs.GetCollection(configs.DB, "auth")

func Register(c *fiber.Ctx) error {
	var user models.Auth
	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Check if user already exists
	count, _ := authCol.CountDocuments(context.TODO(), bson.M{"username": user.Username})
	if count > 0 {
		return c.Status(409).JSON(fiber.Map{"error": "User already exists"})
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	user.Password = string(hash)
	_, err := authCol.InsertOne(context.TODO(), user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to register user"})
	}

	return c.JSON(fiber.Map{"message": "User registered"})
}

func Login(c *fiber.Ctx) error {
	var data models.Auth
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"username": data.Username}).Decode(&user)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "User not found"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(data.Password))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	token, _ := utils.GenerateJWT(user.Username)
	return c.JSON(fiber.Map{"token": token})
}

func UpdatePassword(c *fiber.Ctx) error {
	type Request struct {
		Username        string `json:"username" validate:"required"`
		CurrentPassword string `json:"currentPassword" validate:"required"`
		NewPassword     string `json:"newPassword" validate:"required"`
	}
	var validate = validator.New()

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}
	if err := validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errorMessages := make(map[string]string)
			for _, e := range ve {
				errorMessages[e.Field()] = fmt.Sprintf("failed on '%s' tag", e.Tag())
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"validationErrors": errorMessages})
		}

		// fallback in case it's not a ValidationError
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"username": req.Username}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Current password is incorrect"})
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 14)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash new password"})
	}

	// Update password in DB
	update := bson.M{"$set": bson.M{"password": string(hashedPassword)}}
	_, err = authCol.UpdateOne(context.TODO(), bson.M{"username": req.Username}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
	}

	return c.JSON(fiber.Map{"message": "Password updated successfully"})
}

func ForgotPassword(c *fiber.Ctx) error {
	var req structure.ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed"})
	}

	// Check if user exists
	var user models.User
	err := userCollection.FindOne(context.TODO(), bson.M{"username": req.Username}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	fmt.Print(user)

	// Generate token
	token := utils.GenerateResetToken()
	ExpiresAt := time.Now().Add(15 * time.Minute)


	update := bson.M{"$set": bson.M{"token": string(token), "expiresAt": ExpiresAt}}
	_, err = authCol.UpdateOne(context.TODO(), bson.M{"username": req.Username}, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save token"})
	}

	// // Send to RabbitMQ
	// err = SendResetPasswordEmail(req.Email, token)
	// if err != nil {
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to queue email"})
	// }

	return c.JSON(fiber.Map{"message": "Reset link sent to your email"})
}

func ResetPassword(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get token from request parameters
	resetToken := c.Params("token")
	if resetToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Reset token is required",
		})
	}

	// Define request body structure
	type ResetRequest struct {
		Password        string `json:"password" validate:"required,min=8"`
		ConfirmPassword string `json:"confirmPassword" validate:"required,min=8"`
	}

	var req ResetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Password != req.ConfirmPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Passwords do not match",
		})
	}

	// Find user by reset token
	var auth models.Auth // assuming you have this model
	err := authCol.FindOne(ctx, bson.M{"resetToken": resetToken}).Decode(&auth)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired reset token",
		})
	}

	// Check if the token is expired
	if time.Now().After(auth.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Reset token has expired",
		})
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	// Update password and clear the reset token fields
	update := bson.M{
		"$set": bson.M{"password": string(hashedPassword)},
		"$unset": bson.M{
			"resetToken":       "",
			"resetTokenExpiry": "",
		},
	}

	_, err = authCol.UpdateByID(ctx, auth.ID, update)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update password",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password has been reset successfully",
	})
}

package controllers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"user-auth-profile-service/src/configs"
	"user-auth-profile-service/src/models"
	"user-auth-profile-service/src/rabbitmq"
	"user-auth-profile-service/src/responses"
	"user-auth-profile-service/src/structure"
	"user-auth-profile-service/src/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

var (
	authCol  = configs.GetCollection(configs.DB, "auth")
	producer *rabbitmq.Producer
	authValidate = validator.New()
	config = configs.LoadEnv()
)

func init() {
    maxRetries := 3
    retryDelay := time.Second * 5

    for attempt := range make([]int, maxRetries) {
        conn, err := amqp.Dial(config.AmqpURL)
        if err != nil {
            log.Printf("Failed to connect to RabbitMQ (attempt %d/%d): %v", attempt+1, maxRetries, err)
            if attempt < maxRetries-1 {
                time.Sleep(retryDelay)
                continue
            }
            log.Fatalf("Failed to connect to RabbitMQ after %d attempts: %v", maxRetries, err)
        }

        ch, err := conn.Channel()
        if err != nil {
            log.Fatalf("Failed to open channel: %v", err)
        }

        producer, err = rabbitmq.NewProducer(ch, config.QueueName, true)
        if err != nil {
            log.Fatalf("Failed to create producer: %v", err)
        }

        log.Println("✅ Successfully connected to RabbitMQ")
        return
    }
}

func Register(c *fiber.Ctx) error {
	var req structure.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.RespondWithError(c, fiber.StatusBadRequest, utils.ErrCodeBadRequest, "Invalid request format", err)
	}

	// Validate request
	if err := authValidate.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, e := range ve {
				validationErrors[e.Field()] = fmt.Sprintf("validation failed on '%s' tag", e.Tag())
			}
		}
		return utils.RespondWithValidationError(c, validationErrors)
	}

	// Check if user already exists
	count, _ := authCol.CountDocuments(context.TODO(), bson.M{"email": req.Email})
	if count > 0 {
		return utils.RespondWithError(c, fiber.StatusConflict, utils.ErrCodeDuplicate, "Email already registered", nil)
	}

	// Generate OTP
	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	otpExpiry := time.Now().Add(15 * time.Minute)

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		return utils.RespondWithError(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Failed to hash password", err)
	}

	// Create user with unverified status
	user := models.Auth{
		Email:        req.Email,
		Password:     string(hash),
		OTP:         otp,
		OTPExpiresAt: otpExpiry,
		IsVerified:   false,
	}

	_, err = authCol.InsertOne(context.TODO(), user)
	if err != nil {
		return utils.RespondWithError(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Failed to register user", err)
	}

	// Send OTP via email using RabbitMQ
	emailData := structure.EmailData{
		To:       req.Email,
		Subject:  "Verify Your Email",
		Template: "email_verification",
		Data: map[string]string{
			"otp": otp,
		},
	}
	
	if producer != nil {
		err = producer.Publish(context.Background(), emailData)
		if err != nil {
			// If email fails, delete the user and return error
			authCol.DeleteOne(context.TODO(), bson.M{"email": req.Email})
			return utils.RespondWithError(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Failed to send verification email", err)
		}
	} else {
		log.Println("⚠️ RabbitMQ producer not initialized, skipping email notification")
		return utils.RespondWithError(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Email service unavailable", nil)
	}

	return responses.SendSuccessResponse(c, fiber.StatusCreated, "Registration initiated. Please check your email for OTP verification.", fiber.Map{
		"email": req.Email,
	})
}

func VerifyOTP(c *fiber.Ctx) error {
	var req structure.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Validate request
	if err := authValidate.Struct(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Find user by email
	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	// Check if already verified
	if user.IsVerified {
		return c.Status(400).JSON(fiber.Map{"error": "Email already verified"})
	}

	// Verify OTP
	if user.OTP != req.OTP {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid OTP"})
	}

	// Check OTP expiration
	if time.Now().After(user.OTPExpiresAt) {
		return c.Status(400).JSON(fiber.Map{"error": "OTP has expired"})
	}

	// Update user as verified
	update := bson.M{
		"$set": bson.M{
			"isVerified": true,
			"otp":        "",
			"otpExpiresAt": time.Time{},
		},
	}

	_, err = authCol.UpdateOne(context.TODO(), bson.M{"email": req.Email}, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to verify user"})
	}

	return c.JSON(fiber.Map{"message": "Email verified successfully"})
}

func Login(c *fiber.Ctx) error {
	var data models.Auth
	if err := c.BodyParser(&data); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"email": data.Email}).Decode(&user)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "User not found"})
	}

	// Check if user is verified
	if !user.IsVerified {
		return c.Status(401).JSON(fiber.Map{"error": "Email not verified"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(data.Password))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	token, _ := utils.GenerateJWT(user.Email)
	return c.JSON(fiber.Map{"token": token})
}

func UpdatePassword(c *fiber.Ctx) error {
	type Request struct {
		Email           string `json:"email" validate:"required,email"`
		CurrentPassword string `json:"currentPassword" validate:"required"`
		NewPassword     string `json:"newPassword" validate:"required,min=8"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}
	if err := authValidate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errorMessages := make(map[string]string)
			for _, e := range ve {
				errorMessages[e.Field()] = fmt.Sprintf("failed on '%s' tag", e.Tag())
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"validationErrors": errorMessages})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
	}

	// Check if user is verified
	if !user.IsVerified {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email not verified"})
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
	_, err = authCol.UpdateOne(context.TODO(), bson.M{"email": req.Email}, update)
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

	if err := authValidate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed"})
	}

	// Check if user exists
	var user models.User
	err := userCollection.FindOne(context.TODO(), bson.M{"username": req.Email}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Generate token
	token := utils.GenerateResetToken()
	ExpiresAt := time.Now().Add(15 * time.Minute)


	update := bson.M{"$set": bson.M{"token": string(token), "expiresAt": ExpiresAt}}
	_, err = authCol.UpdateOne(context.TODO(), bson.M{"email": req.Email}, update)
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

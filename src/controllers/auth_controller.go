package controllers

import (
	"context"
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
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid request format", map[string]string{"error": err.Error()})
	}

	// Validate request
	if err := authValidate.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, e := range ve {
				validationErrors[e.Field()] = fmt.Sprintf("validation failed on '%s' tag", e.Tag())
			}
		}
		return responses.SendValidationError(c, validationErrors)
	}

	// Check if user already exists
	count, _ := authCol.CountDocuments(context.TODO(), bson.M{"email": req.Email})
	if count > 0 {
		return responses.SendErrorResponse(c, fiber.StatusConflict, responses.ErrCodeDuplicate, "Email already registered", nil)
	}

	// Generate OTP
	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	otpExpiry := time.Now().Add(15 * time.Minute)

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to hash password", map[string]string{"error": err.Error()})
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
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to register user", map[string]string{"error": err.Error()})
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
			return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to send verification email", map[string]string{"error": err.Error()})
		}
	} else {
		log.Println("⚠️ RabbitMQ producer not initialized, skipping email notification")
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Email service unavailable", nil)
	}

	return responses.SendSuccessResponse(c, fiber.StatusCreated, "Registration initiated. Please check your email for OTP verification.", fiber.Map{
		"email": req.Email,
	})
}

func VerifyOTP(c *fiber.Ctx) error {
	var req structure.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid request format", map[string]string{"error": err.Error()})
	}

	// Validate request
	if err := authValidate.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, e := range ve {
				validationErrors[e.Field()] = fmt.Sprintf("validation failed on '%s' tag", e.Tag())
			}
		}
		return responses.SendValidationError(c, validationErrors)
	}

	// Find user by email
	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusNotFound, responses.ErrCodeNotFound, "User not found", nil)
	}

	// Check if already verified
	if user.IsVerified {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Email already verified", nil)
	}

	// Verify OTP
	if user.OTP != req.OTP {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid OTP", nil)
	}

	// Check OTP expiration
	if time.Now().After(user.OTPExpiresAt) {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "OTP has expired", nil)
	}

	// Update user as verified
	update := bson.M{
		"$set": bson.M{
			"isVerified":   true,
			"otp":          "",
			"otpExpiresAt": time.Time{},
		},
	}

	_, err = authCol.UpdateOne(context.TODO(), bson.M{"email": req.Email}, update)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to verify user", map[string]string{"error": err.Error()})
	}

	return responses.SendSuccessResponse(c, fiber.StatusOK, "Email verified successfully", nil)
}

func Login(c *fiber.Ctx) error {
	var data models.Auth
	if err := c.BodyParser(&data); err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid request format", map[string]string{"error": err.Error()})
	}

	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"email": data.Email}).Decode(&user)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusUnauthorized, responses.ErrCodeUnauthorized, "User not found", nil)
	}

	// Check if user is verified
	if !user.IsVerified {
		return responses.SendErrorResponse(c, fiber.StatusUnauthorized, responses.ErrCodeUnauthorized, "Email not verified", nil)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(data.Password))
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusUnauthorized, responses.ErrCodeUnauthorized, "Invalid credentials", nil)
	}

	token, _ := utils.GenerateJWT(user.Email)
	return responses.SendSuccessResponse(c, fiber.StatusOK, "Login successful", fiber.Map{"token": token})
}

func UpdatePassword(c *fiber.Ctx) error {
	var req structure.UpdatePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid request format", map[string]string{"error": err.Error()})
	}
	if err := authValidate.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, e := range ve {
				validationErrors[e.Field()] = fmt.Sprintf("validation failed on '%s' tag", e.Tag())
			}
		}
		return responses.SendValidationError(c, validationErrors)
	}

	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusUnauthorized, responses.ErrCodeUnauthorized, "User not found", nil)
	}

	// Check if user is verified
	if !user.IsVerified {
		return responses.SendErrorResponse(c, fiber.StatusUnauthorized, responses.ErrCodeUnauthorized, "Email not verified", nil)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword))
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusUnauthorized, responses.ErrCodeUnauthorized, "Current password is incorrect", nil)
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 14)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to hash new password", map[string]string{"error": err.Error()})
	}

	// Update password in DB
	update := bson.M{"$set": bson.M{"password": string(hashedPassword)}}
	_, err = authCol.UpdateOne(context.TODO(), bson.M{"email": req.Email}, update)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to update password", map[string]string{"error": err.Error()})
	}

	return responses.SendSuccessResponse(c, fiber.StatusOK, "Password updated successfully", nil)
}

func ForgotPassword(c *fiber.Ctx) error {
	var req structure.ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid request format", map[string]string{"error": err.Error()})
	}

	if err := authValidate.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, e := range ve {
				validationErrors[e.Field()] = fmt.Sprintf("validation failed on '%s' tag", e.Tag())
			}
		}
		return responses.SendValidationError(c, validationErrors)
	}

	// Check if user exists
	var user models.Auth
	err := authCol.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusNotFound, responses.ErrCodeNotFound, "User not found", nil)
	}

	// Generate token
	rawToken := utils.GenerateResetToken()
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(rawToken), bcrypt.DefaultCost)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to generate reset token", map[string]string{"error": err.Error()})
	}
	ExpiresAt := time.Now().Add(15 * time.Minute)

	update := bson.M{"$set": bson.M{"token": string(hashedToken), "expiresAt": ExpiresAt}}
	_, err = authCol.UpdateOne(context.TODO(), bson.M{"email": req.Email}, update)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to save token", map[string]string{"error": err.Error()})
	}
	emailData := structure.EmailData{
		To:       req.Email,
		Subject:  "Reset Password Token",
		Template: "reset_password",
		Data: map[string]string{
			"token": rawToken,
		},
	}
	if producer != nil {
		err = producer.Publish(context.Background(), emailData)
		if err != nil {
			// If email fails, clear the reset token fields in DB
			resetUpdate := bson.M{
				"$unset": bson.M{
					"token":       "",
					"expiresAt": "",
				},
			}
			authCol.UpdateOne(context.TODO(), bson.M{"email": req.Email}, resetUpdate)
			return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to send verification email", map[string]string{"error": err.Error()})
		}
	} else {
		// If producer is not initialized, clear the reset token fields in DB
		resetUpdate := bson.M{
			"$unset": bson.M{
				"token":       "",
				"expiresAt": "",
			},
		}
		authCol.UpdateOne(context.TODO(), bson.M{"email": req.Email}, resetUpdate)
		log.Println("⚠️ RabbitMQ producer not initialized, skipping email notification")
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Email service unavailable", nil)
	}

	return responses.SendSuccessResponse(c, fiber.StatusOK, "Reset Token sent to your email", nil)
}

func ResetPassword(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get email, token, password from request body
	var req structure.ResetRequest
	if err := c.BodyParser(&req); err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid request format", map[string]string{"error": err.Error()})
	}

	if req.Password != req.ConfirmPassword {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Passwords do not match", nil)
	}

	if req.Email == "" || req.Token == "" {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Email and token are required", nil)
	}

	// Find user by email
	var user models.Auth
	err := authCol.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid or expired reset token", nil)
	}

	// Check if reset token exists and is not expired
	if user.Token == "" || time.Now().After(user.ExpiresAt) {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Reset token is invalid or expired", nil)
	}

	// Compare the provided token with the hashed token in DB
	if err := bcrypt.CompareHashAndPassword([]byte(user.Token), []byte(req.Token)); err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Reset token is invalid", nil)
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to hash password", map[string]string{"error": err.Error()})
	}

	// Update password and clear the reset token fields
	update := bson.M{
		"$set": bson.M{"password": string(hashedPassword)},
		"$unset": bson.M{
			"token":       "",
			"expiresAt": "",
		},
	}

	_, err = authCol.UpdateOne(ctx, bson.M{"email": req.Email}, update)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to update password", map[string]string{"error": err.Error()})
	}

	return responses.SendSuccessResponse(c, fiber.StatusOK, "Password has been reset successfully", nil)
}

package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"user-auth-profile-service/src/configs"
	"user-auth-profile-service/src/models"
	"user-auth-profile-service/src/responses"
	"user-auth-profile-service/src/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	userCollection *mongo.Collection = configs.GetCollection(configs.DB, "users")
	validate = validator.New()
)

func CreateUser(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	name := c.FormValue("name")
	email := c.FormValue("email")
	location := c.FormValue("location")
	title := c.FormValue("title")
	address := c.FormValue("address")
	linkedin := c.FormValue("linkedin")
	twitter := c.FormValue("twitter")
	dob := c.FormValue("dob")
	fileHeader, err := c.FormFile("resume")
	username := c.Locals("username").(string)

	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Resume file is required", map[string]string{"error": err.Error()})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to open resume file", map[string]string{"error": err.Error()})
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("⚠️ Failed to close file: %v", err)
		}
	}()

	var user models.User
	err = userCollection.FindOne(ctx, bson.M{
		"$or": []bson.M{
			{"email": email},
			{"username": username},
		}}).Decode(&user)
	if err == nil {
		return responses.SendErrorResponse(c, http.StatusConflict, responses.ErrCodeDuplicate, "User already exists", nil)
	}

	s3Client, bucketName := utils.InitS3()
	resumeURL, err := utils.UploadToS3(s3Client, bucketName, file, fileHeader.Filename)
	if err != nil {
		return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to upload resume to S3", map[string]string{"error": err.Error()})
	}

	newUser := models.User{
		Id:       primitive.NewObjectID(),
		Name:     name,
		Email:    email,
		Location: location,
		Title:    title,
		Address:  address,
		LinkedIn: linkedin,
		Twitter:  twitter,
		DOB:      dob,
		Resume:   resumeURL,
		Username: username,
	}

	if validationErr := validate.Struct(&newUser); validationErr != nil {
		validationErrors := make(map[string]string)
		if ve, ok := validationErr.(validator.ValidationErrors); ok {
			for _, e := range ve {
				validationErrors[e.Field()] = fmt.Sprintf("validation failed on '%s' tag", e.Tag())
			}
		}
		return responses.SendValidationError(c, validationErrors)
	}

	result, err := userCollection.InsertOne(ctx, newUser)
	if err != nil {
		return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to save user", map[string]string{"error": err.Error()})
	}

	return responses.SendSuccessResponse(c, http.StatusCreated, "User created successfully", fiber.Map{"data": result})
}

func GetAUser(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	userId := c.Params("userId")
	var user models.User
	defer cancel()

	objId, _ := primitive.ObjectIDFromHex(userId)

	err := userCollection.FindOne(ctx, bson.M{"id": objId}).Decode(&user)
	if err != nil {
		return responses.SendErrorResponse(c, http.StatusNotFound, responses.ErrCodeNotFound, "User does not exist", map[string]string{"error": err.Error()})
	}

	return responses.SendSuccessResponse(c, http.StatusOK, "success", fiber.Map{"data": user})
}

func EditAUser(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userId := c.Params("userId")
	var user models.User

	objId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Invalid user ID", map[string]string{"error": err.Error()})
	}

	if err := c.BodyParser(&user); err != nil {
		return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Failed to parse body", map[string]string{"error": err.Error()})
	}

	if validationErr := validate.Struct(&user); validationErr != nil {
		validationErrors := make(map[string]string)
		if ve, ok := validationErr.(validator.ValidationErrors); ok {
			for _, e := range ve {
				validationErrors[e.Field()] = fmt.Sprintf("validation failed on '%s' tag", e.Tag())
			}
		}
		return responses.SendValidationError(c, validationErrors)
	}

	var resumeURL string
	fileHeader, err := c.FormFile("resume")
	if err == nil && fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			return responses.SendErrorResponse(c, fiber.StatusBadRequest, responses.ErrCodeBadRequest, "Failed to open resume file", map[string]string{"error": err.Error()})
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("⚠️ Failed to close file: %v", err)
			}
		}()

		s3Client, bucketName := utils.InitS3()
		uploadURL, err := utils.UploadToS3(s3Client, bucketName, file, fileHeader.Filename)
		if err != nil {
			return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to upload to S3", map[string]string{"error": err.Error()})
		}
		resumeURL = uploadURL
	}

	update := bson.M{
		"name":     user.Name,
		"location": user.Location,
		"title":    user.Title,
		"address":  user.Address,
		"linkedin": user.LinkedIn,
		"twitter":  user.Twitter,
		"dob":      user.DOB,
	}
	if resumeURL != "" {
		update["resume_url"] = resumeURL
	}

	result, err := userCollection.UpdateOne(ctx, bson.M{"id": objId}, bson.M{"$set": update})
	if err != nil {
		return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to update user", map[string]string{"error": err.Error()})
	}

	var updatedUser models.User
	if result.MatchedCount == 1 {
		err := userCollection.FindOne(ctx, bson.M{"id": objId}).Decode(&updatedUser)
		if err != nil {
			return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to fetch updated user", map[string]string{"error": err.Error()})
		}
	}

	return responses.SendSuccessResponse(c, fiber.StatusOK, "User updated successfully", fiber.Map{"data": updatedUser})
}

func DeleteAUser(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	userId := c.Params("userId")
	defer cancel()

	objId, _ := primitive.ObjectIDFromHex(userId)

	result, err := userCollection.DeleteOne(ctx, bson.M{"id": objId})
	if err != nil {
		return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to delete user", map[string]string{"error": err.Error()})
	}

	if result.DeletedCount < 1 {
		return responses.SendErrorResponse(c, http.StatusNotFound, responses.ErrCodeNotFound, "User with specified ID not found!", nil)
	}

	return responses.SendSuccessResponse(c, http.StatusOK, "User successfully deleted", nil)
}

func DeleteAllUsers(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    result, err := userCollection.DeleteMany(ctx, bson.M{})
    if err != nil {
        return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to delete users", map[string]string{"error": err.Error()})
    }

    if result.DeletedCount < 1 {
        return responses.SendErrorResponse(c, http.StatusNotFound, responses.ErrCodeNotFound, "No users found to delete!", nil)
    }

    return responses.SendSuccessResponse(c, http.StatusOK, "All users successfully deleted", fiber.Map{"count": result.DeletedCount})
}

func GetAllUsers(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var users []models.User
	defer cancel()

	results, err := userCollection.Find(ctx, bson.M{})
	if err != nil {
		return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Failed to fetch users", map[string]string{"error": err.Error()})
	}

	defer func() {
		if err := results.Close(ctx); err != nil {
			log.Printf("⚠️ Failed to close Mongo cursor: %v", err)
		}
	}()

	for results.Next(ctx) {
		var singleUser models.User
		if err = results.Decode(&singleUser); err != nil {
			return responses.SendErrorResponse(c, http.StatusInternalServerError, responses.ErrCodeInternalError, "Error in fetching", map[string]string{"error": err.Error()})
		}
		users = append(users, singleUser)
	}

	return responses.SendSuccessResponse(c, http.StatusOK, "success", fiber.Map{"data": users})
}

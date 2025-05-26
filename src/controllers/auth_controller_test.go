package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"user-auth-profile-service/src/models"
	"user-auth-profile-service/src/responses"
	"user-auth-profile-service/src/structure"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
)

var testAuthCol *mongo.Collection

func setupApp() *fiber.App {
	app := fiber.New()
	app.Post("/auth/register", Register)
	app.Post("/auth/login", Login)
	return app
}

func initTestDB(db *mongo.Database) {
	// Override the authCol with our test collection
	authCol = db.Collection("auth")
	testAuthCol = authCol
}

func TestMain(m *testing.M) {
	db := SetupTestDB(&testing.T{})
	initTestDB(db)
	code := m.Run()
	TearDownTestDB(&testing.T{})
	os.Exit(code)
}

func TestRegister_Success(t *testing.T) {
	app := setupApp()
	// Use random data for isolation
	email := randomEmail()
	password := randomPassword()
	payload := structure.RegisterRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var res responses.Response
	json.NewDecoder(resp.Body).Decode(&res)
	assert.True(t, res.Success)
	assert.Contains(t, res.Message, "Registration initiated")

	user, err := getUserByEmail(email)
	assert.NoError(t, err)
	assert.Equal(t, email, user.Email)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	app := setupApp()
	payload := structure.RegisterRequest{
		Email:    "testuser@example.com",
		Password: "TestPassword123!",
	}
	body, _ := json.Marshal(payload)
	// Register first time
	_, _ = app.Test(httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body)), -1)
	// Register second time (should fail)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	var res responses.Response
	json.NewDecoder(resp.Body).Decode(&res)
	assert.False(t, res.Success)
	assert.Contains(t, res.Message, "Email already registered")
}

func TestLogin_Success(t *testing.T) {
	app := setupApp()
	email := "loginuser@example.com"
	password := "LoginPassword123!"

	// Register user first
	registerPayload := structure.RegisterRequest{
		Email:    email,
		Password: password,
	}
	registerBody, _ := json.Marshal(registerPayload)
	registerResp, err := app.Test(httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody)), -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, registerResp.StatusCode, "Registration should succeed")

	// Verify user exists in DB
	user, err := getUserByEmail(email)
	assert.NoError(t, err, "User should exist in DB after registration")
	assert.Equal(t, email, user.Email)
	fmt.Printf("User in DB: %+v\n", user) // Debug print

	loginPayload := models.Auth{
		Email:    email,
		Password: password,
	}
	loginBody, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Read and print response body for debugging
	var res responses.Response
	json.NewDecoder(resp.Body).Decode(&res)
	fmt.Printf("Login Response: %+v\n", res) // Debug print

	// Expect 401 since user is not verified
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.False(t, res.Success)
	assert.Contains(t, res.Message, "Email not verified")
}

func TestLogin_InvalidCredentials(t *testing.T) {
	app := setupApp()
	loginPayload := models.Auth{
		Email:    "nonexistent@example.com",
		Password: "WrongPassword!",
	}
	loginBody, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var res responses.Response
	json.NewDecoder(resp.Body).Decode(&res)
	assert.False(t, res.Success)
	// Accept either "User not found" or "Invalid credentials" as valid error messages for invalid login
	fmt.Print("res",res)
	assert.Contains(t, res.Message, "User not found")
	// Optionally, if your login returns "Invalid credentials" for wrong password, you can use:
	// assert.Contains(t, res.Message, "Invalid credentials")
}

func randomEmail() string {
	return "user" + strconv.Itoa(rand.Intn(1000000)) + "@example.com"
}

func randomPassword() string {
	return "Pass" + strconv.Itoa(rand.Intn(1000000)) + "!@#"
}

// Helper to fetch user from DB by email (for verification in tests)
func getUserByEmail(email string) (*models.Auth, error) {
	user := &models.Auth{}
	err := authCol.FindOne(context.TODO(), map[string]interface{}{"email": email}).Decode(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

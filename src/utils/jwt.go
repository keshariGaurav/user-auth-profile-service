package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"user-auth-profile-service/src/configs"

	"github.com/golang-jwt/jwt/v5"
)


func JWTSecretKey() string {
    return os.Getenv("JWT_SECRET_KEY")
}

var SecretKey = []byte(JWTSecretKey())

type JWTClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateJWT(email string) (string, error) {
	config := configs.LoadEnv()
	
	claims := jwt.MapClaims{
		"email":  email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
		"iss":   config.JWTIssuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("cannot parse claims")
	}

	return claims, nil
}

func GenerateResetToken() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}


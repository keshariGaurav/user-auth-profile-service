package structure

import (
	"time"
)
type ResetToken struct {
	Email     string    `bson:"email"`
	Token     string    `bson:"token"`
	ExpiresAt time.Time `bson:"expiresAt"`
}

type ResetPassword struct {
	Password     string    				`bson:"password"`
	ConfirmPassword     string    `bson:"confirmPassword"`
}
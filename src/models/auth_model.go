package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Auth struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email        string             `bson:"email" json:"email" validate:"required,email"`
	Password     string             `bson:"password" json:"password" validate:"required,min=8"`
	IsVerified   bool               `bson:"isVerified" json:"isVerified"`
	OTP          string             `bson:"otp,omitempty" json:"otp,omitempty"`
	OTPExpiresAt time.Time          `bson:"otpExpiresAt,omitempty" json:"otpExpiresAt,omitempty"`
	Token        string             `bson:"token,omitempty" json:"token,omitempty"`
	ExpiresAt    time.Time          `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
}

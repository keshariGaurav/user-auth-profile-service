package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Auth struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Username string             `bson:"username" json:"username" validate:"required"`
	Password string             `bson:"password" json:"password" validate:"required,min=8"`
	Token 	 string             `bson:"token" json:"token"`
	ExpiresAt time.Time 				`bson:"expiresAt"`
}

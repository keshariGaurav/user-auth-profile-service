package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Auth struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Username string             `bson:"username" json:"username" validate:"required"`
	Password string             `bson:"password" json:"password" validate:"required,min=8"`
}
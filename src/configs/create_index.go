package configs

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SetupUserIndexes(collection *mongo.Collection) error {
	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	// Create all indexes
	_, err := collection.Indexes().CreateMany(context.TODO(), indexModels)
	if err != nil {
		return fmt.Errorf("failed to create user indexes: %w", err)
	}

	log.Println("✅ All user indexes created successfully")
	return nil
}

func SetupAllIndexes() error {
	var userCol = GetCollection(DB, "users")
	if err := SetupUserIndexes(userCol); err != nil {
		return fmt.Errorf("failed to setup user indexes: %w", err)
	}
	return nil
}

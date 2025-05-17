package configs

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB() *mongo.Client {
	config := LoadEnv()
	maxRetries := 3
	retryDelay := time.Second * 5

	for attempt := range make([]int, maxRetries) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		serverAPI := options.ServerAPI(options.ServerAPIVersion1)
		clientOpts := options.Client().ApplyURI(config.MongoURI).SetServerAPIOptions(serverAPI)

		client, err := mongo.Connect(ctx, clientOpts)
		if err != nil {
			log.Printf("❌ Failed to connect to MongoDB (attempt %d/%d): %v", attempt+1, maxRetries, err)
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay)
				continue
			}
			log.Fatalf("❌ Failed to connect to MongoDB after %d attempts: %v", maxRetries, err)
		}

		// Ping the database
		if err := client.Ping(ctx, nil); err != nil {
			log.Printf("❌ MongoDB ping failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay)
				continue
			}
			log.Fatalf("❌ MongoDB ping failed after %d attempts: %v", maxRetries, err)
		}

		log.Printf("✅ Connected to MongoDB successfully on attempt %d/%d", attempt+1, maxRetries)
		return client
	}
	return nil
}

// Global DB client instance
var DB *mongo.Client = ConnectDB()

func init() {
	// Setup indexes after DB connection is established
	if err := SetupAllIndexes(); err != nil {
		log.Fatalf("Failed to setup indexes: %v", err)
	}
}

// GetCollection returns a MongoDB collection by name
func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	return client.Database("golangAPI").Collection(collectionName)
}

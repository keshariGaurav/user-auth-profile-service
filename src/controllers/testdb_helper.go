package controllers

import (
	"context"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var testMongoClient *mongo.Client
var testDBName = "user-auth-profile-test"

// SetupTestDB connects to the test database and drops all collections before each test run.
func SetupTestDB(t *testing.T) *mongo.Database {
	uri := os.Getenv("MONGOURI")
	if uri == "" {
		uri = "mongodb://localhost:27017" // fallback for local dev
	}
	clientOpts := options.Client().ApplyURI(uri)
	var err error
	testMongoClient, err = mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	db := testMongoClient.Database(testDBName)
	// Drop all collections
	collections, err := db.ListCollectionNames(context.Background(), struct{}{})
	if err != nil {
		t.Fatalf("Failed to list collections: %v", err)
	}
	for _, coll := range collections {
		_ = db.Collection(coll).Drop(context.Background())
	}
	return db
}

// TearDownTestDB disconnects the test client.
func TearDownTestDB(t *testing.T) {
	if testMongoClient != nil {
		_ = testMongoClient.Disconnect(context.Background())
	}
}

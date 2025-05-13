package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env           string
	AmqpURL       string
	QueueName     string
	// MongoDB Configuration
	MongoURI string

	// AWS Configuration
	AWSBucketName string

	// JWT Configuration
	JWTSecret      string
	JWTIssuer      string
}

func LoadEnv() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, continuing")
	}

	return Config{
		// MongoDB
		MongoURI: os.Getenv("MONGOURI"),
		// RabbitMQ
		Env:         os.Getenv("ENV"),
		AmqpURL:     os.Getenv("AMQP_URL"),
		QueueName:   os.Getenv("QUEUE_NAME"),

		// AWS
		AWSBucketName: os.Getenv("AWS_S3_BUCKET"),

		// JWT
		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTIssuer:      os.Getenv("JWT_ISSUER"),
	}
}

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
	JWTSecretKey      string
	JWTExpirationTime string
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
		JWTSecretKey:      os.Getenv("JWT_SECRET_KEY"),
		JWTExpirationTime: os.Getenv("JWT_EXPIRATION_TIME"),
	}
}

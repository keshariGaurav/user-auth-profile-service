package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Producer struct {
	channel   *amqp.Channel
	queueName string
}

// NewProducer initializes a new Producer
func NewProducer(ch *amqp.Channel, queueName string, durable bool) (*Producer, error) {
	log.Printf("✅ Producer initialized for queue [%s]", queueName)
	return &Producer{
		channel:   ch,
		queueName: queueName,
	}, nil
}

// Publish sends a JSON-encoded message to the queue
func (p *Producer) Publish(ctx context.Context, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		"",          // exchange (empty for direct queue publishing)
		p.queueName, // routing key (queue name)
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}

	log.Printf("📤 Message published to queue [%s]", p.queueName)
	return nil
}

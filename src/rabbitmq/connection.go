package rabbitmq

import (
	"context"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	mu         sync.Mutex
	Conn       *amqp.Connection
	Channel    *amqp.Channel
	amqpURL    string
	queueName  string        // Add queue name field
	notifyConn chan *amqp.Error
	notifyChan chan *amqp.Error
	ctx        context.Context
	cancel     context.CancelFunc
}

var (
	instance *Connection
	once     sync.Once
)

// NewConnection ensures singleton pattern with reconnect logic
func NewConnection(amqpURL string) (*Connection, error) {
	var err error
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		conn := &Connection{
			amqpURL:    amqpURL,
			queueName:  "email_queue", // Set default queue name
			ctx:        ctx,
			cancel:     cancel,
		}
		err = conn.connect()
		if err == nil {
			instance = conn
			go instance.reconnectOnFailure()
		}
	})
	return instance, err
}

func (c *Connection) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := amqp.Dial(c.amqpURL)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	// Declare the queue
	_, err = ch.QueueDeclare(
		c.queueName, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	c.Conn = conn
	c.Channel = ch
	c.notifyConn = conn.NotifyClose(make(chan *amqp.Error))
	c.notifyChan = ch.NotifyClose(make(chan *amqp.Error))

	log.Println("✅ RabbitMQ connected and channel opened.")
	return nil
}

// reconnectOnFailure watches for closures and reconnects automatically
func (c *Connection) reconnectOnFailure() {
	for {
		select {
		case <-c.ctx.Done():
			log.Println("🛑 Stopping reconnect goroutine.")
			return
		case err := <-c.notifyConn:
			log.Printf("🚨 RabbitMQ connection closed: %v. Reconnecting...", err)
			c.reconnect()
		case err := <-c.notifyChan:
			log.Printf("🚨 RabbitMQ channel closed: %v. Reconnecting...", err)
			c.reconnect()
		}
	}
}

func (c *Connection) reconnect() {
	var wait time.Duration = time.Second
	for {
		err := c.connect()
		if err == nil {
			return
		}
		log.Printf("Reconnection failed: %v. Retrying in %v...", err, wait)
		time.Sleep(wait)
		wait *= 2
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
	}
}

func (c *Connection) Close() {
	if c.cancel != nil {
		c.cancel() // Cancel context to stop reconnect goroutine
	}
	if c.Channel != nil {
		_ = c.Channel.Close()
	}
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
	instance = nil
}

// Add this method after the existing methods
func (c *Connection) GetQueueName() string {
	return c.queueName
}

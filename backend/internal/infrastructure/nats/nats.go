package nats

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// Client wraps NATS JetStream for event publishing and subscribing.
type Client struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	stream *nats.StreamInfo
}

// NewClient connects to NATS and initializes the JetStream stream.
func NewClient(url string) (*Client, error) {
	nc, err := nats.Connect(url,
		nats.Timeout(5*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to init JetStream: %w", err)
	}

	// Create stream if not exists
	streamName := "BERTH_EVENTS"
	stream, err := js.StreamInfo(streamName)
	if err != nil {
		stream, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"berth.>", "sandbox.>", "file.>"},
			Storage:  nats.FileStorage,
			MaxAge:   7 * 24 * time.Hour,
		})
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("failed to create stream: %w", err)
		}
		slog.Info("created NATS stream", "name", streamName)
	}

	return &Client{
		conn:   nc,
		js:     js,
		stream: stream,
	}, nil
}

// Publish sends a message to a subject.
func (c *Client) Publish(subject string, data []byte) error {
	_, err := c.js.Publish(subject, data)
	return err
}

// Subscribe creates a durable consumer subscription.
func (c *Client) Subscribe(subject, durable string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.js.Subscribe(subject, handler,
		nats.Durable(durable),
		nats.ManualAck(),
		nats.MaxDeliver(3),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	return sub, nil
}

// Close closes the NATS connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

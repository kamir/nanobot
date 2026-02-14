package group

import (
	"context"
	"log/slog"
	"strings"

	"github.com/segmentio/kafka-go"
)

// KafkaConsumer implements the Consumer interface using segmentio/kafka-go.
type KafkaConsumer struct {
	brokers       string
	consumerGroup string
	topics        []string
	readers       []*kafka.Reader
	messages      chan ConsumerMessage
}

// NewKafkaConsumer creates a Kafka consumer for the given topics.
func NewKafkaConsumer(brokers, consumerGroup string, topics []string) *KafkaConsumer {
	return &KafkaConsumer{
		brokers:       brokers,
		consumerGroup: consumerGroup,
		topics:        topics,
		messages:      make(chan ConsumerMessage, 100),
	}
}

// Start begins consuming from all configured topics.
func (c *KafkaConsumer) Start(ctx context.Context) error {
	brokerList := strings.Split(c.brokers, ",")

	for _, topic := range c.topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokerList,
			Topic:    topic,
			GroupID:  c.consumerGroup,
			MinBytes: 1,
			MaxBytes: 10e6,
		})
		c.readers = append(c.readers, reader)

		go func(r *kafka.Reader, t string) {
			for {
				msg, err := r.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					slog.Warn("KafkaConsumer: read error", "topic", t, "error", err)
					continue
				}
				c.messages <- ConsumerMessage{
					Topic: t,
					Key:   msg.Key,
					Value: msg.Value,
				}
			}
		}(reader, topic)
	}

	return nil
}

// Messages returns the channel of consumed messages.
func (c *KafkaConsumer) Messages() <-chan ConsumerMessage {
	return c.messages
}

// Close stops all readers.
func (c *KafkaConsumer) Close() error {
	for _, r := range c.readers {
		r.Close()
	}
	close(c.messages)
	return nil
}

// ChannelConsumer is a test/in-process Consumer implementation backed by a Go channel.
type ChannelConsumer struct {
	ch chan ConsumerMessage
}

// NewChannelConsumer creates an in-process consumer for testing.
func NewChannelConsumer() *ChannelConsumer {
	return &ChannelConsumer{
		ch: make(chan ConsumerMessage, 100),
	}
}

// Start is a no-op for the channel consumer.
func (c *ChannelConsumer) Start(ctx context.Context) error { return nil }

// Messages returns the message channel.
func (c *ChannelConsumer) Messages() <-chan ConsumerMessage { return c.ch }

// Close closes the channel.
func (c *ChannelConsumer) Close() error {
	close(c.ch)
	return nil
}

// Send pushes a message into the channel consumer (for testing).
func (c *ChannelConsumer) Send(msg ConsumerMessage) {
	c.ch <- msg
}

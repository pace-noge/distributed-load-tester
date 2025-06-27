package kafka

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaConsumer implements the domain.KafkaConsumer interface.
type KafkaConsumer struct {
	reader *kafka.Reader
}

// NewKafkaConsumer creates a new Kafka consumer.
func NewKafkaConsumer(brokerAddress, topic, groupID string) (*KafkaConsumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{brokerAddress},
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3,        // 10KB
		MaxBytes:       10e6,        // 10MB
		CommitInterval: time.Second, // Flush commits to Kafka every second
		MaxAttempts:    3,
		Dialer:         &kafka.Dialer{Timeout: 10 * time.Second}, // Add timeout for dialer
	})
	log.Printf("Kafka consumer initialized for topic %s, group %s at %s", topic, groupID, brokerAddress)
	return &KafkaConsumer{reader: reader}, nil
}

// Consume starts consuming messages from Kafka. The handler function will be called for each message.
func (kc *KafkaConsumer) Consume(ctx context.Context, topic string, handler func(key, value []byte) error) error {
	log.Printf("Starting Kafka consumer for topic: %s", topic)
	for {
		m, err := kc.reader.FetchMessage(ctx)
		if err != nil {
			// Handle context cancellation
			if ctx.Err() == context.Canceled {
				log.Println("Kafka consumer context cancelled. Shutting down.")
				return ctx.Err()
			}
			log.Printf("Error fetching Kafka message: %v", err)
			time.Sleep(time.Second) // Small backoff before retrying
			continue
		}

		log.Printf("Received message from partition %d, offset %d: %s = %s\n", m.Partition, m.Offset, string(m.Key), string(m.Value))

		// Process message with handler
		err = handler(m.Key, m.Value)
		if err != nil {
			log.Printf("Error processing message (key: %s, topic: %s): %v. Not committing offset.", string(m.Key), m.Topic, err)
			// Depending on business logic, you might want to Nack the message or retry.
			// For simplicity, we just log and continue, the message will be re-fetched next time if not committed.
		} else {
			// Commit the offset only if processing was successful
			err = kc.reader.CommitMessages(ctx, m)
			if err != nil {
				log.Printf("Error committing Kafka offset: %v", err)
			}
		}
	}
}

// Close closes the Kafka consumer.
func (kc *KafkaConsumer) Close() error {
	log.Println("Closing Kafka consumer...")
	return kc.reader.Close()
}

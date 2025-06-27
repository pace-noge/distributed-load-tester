package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer implements the domain.KafkaProducer interface.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a new Kafka producer.
func NewKafkaProducer(brokerAddress, topic string) (*KafkaProducer, error) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		// Optional: configure retries, timeouts, batching
		WriteTimeout: time.Second * 10,
		RequiredAcks: kafka.RequireOne, // Ensure at least one replica has acknowledged write
	}
	log.Printf("Kafka producer initialized for topic %s at %s", topic, brokerAddress)
	return &KafkaProducer{writer: writer}, nil
}

// Produce sends a message to Kafka using the pre-configured topic.
func (kp *KafkaProducer) Produce(ctx context.Context, key string, value []byte) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: value,
	}
	err := kp.writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to write kafka message: %w", err)
	}
	log.Printf("Produced message with key '%s'", key)
	return nil
}

// Close closes the Kafka producer.
func (kp *KafkaProducer) Close() error {
	log.Println("Closing Kafka producer...")
	return kp.writer.Close()
}

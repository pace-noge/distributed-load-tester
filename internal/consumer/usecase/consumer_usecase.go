package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
)

// ConsumerUsecase handles the business logic for the result consumer.
type ConsumerUsecase struct {
	testResultRepo       domain.TestResultRepository
	aggregatedResultRepo domain.AggregatedResultRepository
	kafkaConsumer        domain.KafkaConsumer
}

// NewConsumerUsecase creates a new ConsumerUsecase instance.
func NewConsumerUsecase(trr domain.TestResultRepository, arr domain.AggregatedResultRepository, kc domain.KafkaConsumer) *ConsumerUsecase {
	return &ConsumerUsecase{
		testResultRepo:       trr,
		aggregatedResultRepo: arr,
		kafkaConsumer:        kc,
	}
}

// StartConsuming begins consuming messages from the specified Kafka topic.
func (uc *ConsumerUsecase) StartConsuming(ctx context.Context, topic string) error {
	return uc.kafkaConsumer.Consume(ctx, topic, uc.handleKafkaMessage)
}

// handleKafkaMessage processes each message received from Kafka.
func (uc *ConsumerUsecase) handleKafkaMessage(key, value []byte) error {
	log.Printf("Consumer received message: Key=%s, Value_Length=%d", string(key), len(value))

	var result domain.TestResult
	err := json.Unmarshal(value, &result)
	if err != nil {
		log.Printf("Error unmarshalling Kafka message to TestResult: %v, Value: %s", err, string(value))
		return fmt.Errorf("failed to unmarshal message: %w", err) // Return error to prevent committing offset for bad message
	}

	// Persist the raw test result
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = uc.testResultRepo.SaveTestResult(ctx, &result)
	if err != nil {
		log.Printf("Error saving raw test result for test %s, worker %s: %v", result.TestID, result.WorkerID, err)
		return fmt.Errorf("failed to save raw test result: %w", err)
	}

	log.Printf("Saved raw result for Test ID: %s, Worker ID: %s", result.TestID, result.WorkerID)

	// Aggregation logic: This can be more complex, potentially triggering aggregation
	// only when all workers for a specific test have reported, or on a schedule.
	// For this example, we'll assume master handles final aggregation upon worker completion.
	// The consumer's primary role here is to persist raw data.
	// If you want the consumer to trigger aggregation as well, you'd add logic here
	// to check if a test is complete and then call a function to aggregate.

	return nil
}

package config

import (
	"log"

	"github.com/spf13/viper"
)

// ConsumerConfig holds configuration for the Result Consumer service.
type ConsumerConfig struct {
	KafkaBroker string `mapstructure:"KAFKA_BROKER"`
	KafkaTopic  string `mapstructure:"KAFKA_TOPIC"` // Topic to consume results from
	KafkaGroup  string `mapstructure:"KAFKA_GROUP"` // Consumer group ID
	DatabaseURL string `mapstructure:"DATABASE_URL"`
}

// LoadConsumerConfig loads consumer service configuration from environment variables or config file.
func LoadConsumerConfig() (*ConsumerConfig, error) {
	viper.SetConfigFile(".env") // Look for .env file
	viper.AddConfigPath(".")
	viper.AutomaticEnv() // Read from environment variables

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Warning: No .env file found, reading config from environment variables only: %v", err)
	}

	cfg := &ConsumerConfig{
		KafkaBroker: "localhost:9092",
		KafkaTopic:  "test_results",
		KafkaGroup:  "load_tester_consumer_group", // Unique group ID for consumers
		DatabaseURL: "postgres://user:password@localhost:5432/distributed_load_tester?sslmode=disable",
	}

	// Override with values from Viper
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

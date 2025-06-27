package config

import (
	"log"

	"github.com/spf13/viper"
)

// WorkerConfig holds configuration for the Worker service.
type WorkerConfig struct {
	GRPCPort      int    `mapstructure:"GRPC_PORT"`
	MasterAddress string `mapstructure:"MASTER_ADDRESS"` // Master gRPC address (host:port)
	KafkaBroker   string `mapstructure:"KAFKA_BROKER"`
	KafkaTopic    string `mapstructure:"KAFKA_TOPIC"` // Topic to produce results to
	WorkerID      string `mapstructure:"WORKER_ID"`   // Unique ID for this worker
}

// LoadWorkerConfig loads worker service configuration from environment variables or config file.
func LoadWorkerConfig() (*WorkerConfig, error) {
	viper.SetConfigFile(".env") // Look for .env file
	viper.AddConfigPath(".")
	viper.AutomaticEnv() // Read from environment variables

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Warning: No .env file found, reading config from environment variables only: %v", err)
	}

	cfg := &WorkerConfig{
		GRPCPort:      50052, // Worker can also expose gRPC if needed for future features
		MasterAddress: "localhost:50051",
		KafkaBroker:   "localhost:9092",
		KafkaTopic:    "test_results",
		WorkerID:      "worker-1", // Default, but should be unique in deployment
	}

	// Override with values from Viper
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

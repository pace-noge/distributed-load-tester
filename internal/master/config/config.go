package config

import (
	"log"

	"github.com/spf13/viper"
)

// MasterConfig holds configuration for the Master service.
type MasterConfig struct {
	GRPCPort     int    `mapstructure:"GRPC_PORT"`
	HTTPPort     int    `mapstructure:"HTTP_PORT"`
	KafkaBroker  string `mapstructure:"KAFKA_BROKER"`
	KafkaTopic   string `mapstructure:"KAFKA_TOPIC"` // Topic for worker results
	DatabaseURL  string `mapstructure:"DATABASE_URL"`
	JWTSecretKey string `mapstructure:"JWT_SECRET_KEY"`
}

// LoadMasterConfig loads master service configuration from environment variables or config file.
func LoadMasterConfig() (*MasterConfig, error) {
	viper.SetConfigFile(".env") // Look for .env file
	viper.AddConfigPath(".")
	viper.AutomaticEnv() // Read from environment variables

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Warning: No .env file found, reading config from environment variables only: %v", err)
	}

	cfg := &MasterConfig{
		GRPCPort:     50051,
		HTTPPort:     8080,
		KafkaBroker:  "localhost:9092",
		KafkaTopic:   "test_results",
		DatabaseURL:  "postgres://postgres:password@localhost:5432/distributed_load_tester?sslmode=disable",
		JWTSecretKey: "your-very-secret-key-that-should-be-in-env", // Default, but override with env var
	}

	// Override with values from Viper
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// Ensure JWT_SECRET_KEY is set
	if cfg.JWTSecretKey == "" || cfg.JWTSecretKey == "your-very-secret-key-that-should-be-in-env" {
		log.Println("WARNING: JWT_SECRET_KEY is not set or using default. Please set a strong, unique key in production.")
	}

	return cfg, nil
}

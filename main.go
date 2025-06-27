// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	// Import all internal packages
	consumerUsecase "github.com/pace-noge/distributed-load-tester/internal/consumer/usecase" // Renamed to avoid conflict with `usecase` directory
	"github.com/pace-noge/distributed-load-tester/internal/domain"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/auth"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/database"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/kafka"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/vegeta"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/worker_repo"
	masterGRPC "github.com/pace-noge/distributed-load-tester/internal/master/delivery/grpc"
	masterHTTP "github.com/pace-noge/distributed-load-tester/internal/master/delivery/http"
	masterUsecase "github.com/pace-noge/distributed-load-tester/internal/master/usecase" // Renamed to avoid conflict with `usecase` directory
	workerGRPC "github.com/pace-noge/distributed-load-tester/internal/worker/delivery/grpc"
	workerUsecase "github.com/pace-noge/distributed-load-tester/internal/worker/usecase" // Renamed to avoid conflict with `usecase` directory
	pb "github.com/pace-noge/distributed-load-tester/proto"
)

func main() {
	app := &cli.App{
		Name:  "load-tester-app",
		Usage: "A distributed load testing application (master, worker, or consumer).",
		Commands: []*cli.Command{
			{
				Name:  "master",
				Usage: "Starts the Master service",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "grpc-port",
						Aliases: []string{"gp"},
						Value:   50051,
						Usage:   "gRPC port for Master service",
						EnvVars: []string{"GRPC_PORT"},
					},
					&cli.IntFlag{
						Name:    "http-port",
						Aliases: []string{"hp"},
						Value:   8080,
						Usage:   "HTTP port for Master service (UI and API)",
						EnvVars: []string{"HTTP_PORT"},
					},
					&cli.StringFlag{
						Name:    "kafka-broker",
						Aliases: []string{"kb"},
						Value:   "localhost:9092",
						Usage:   "Kafka broker address",
						EnvVars: []string{"KAFKA_BROKER"},
					},
					&cli.StringFlag{
						Name:    "kafka-topic",
						Aliases: []string{"kt"},
						Value:   "test_results",
						Usage:   "Kafka topic for worker results",
						EnvVars: []string{"KAFKA_TOPIC"},
					},
					&cli.StringFlag{
						Name:    "database-url",
						Aliases: []string{"db"},
						Value:   "postgres://user:password@localhost:5432/distributed_load_tester?sslmode=disable",
						Usage:   "PostgreSQL database connection URL",
						EnvVars: []string{"DATABASE_URL"},
					},
					&cli.StringFlag{
						Name:    "jwt-secret-key",
						Aliases: []string{"jwt"},
						Value:   "your-very-secret-key-that-should-be-in-env",
						Usage:   "JWT secret key for authentication",
						EnvVars: []string{"JWT_SECRET_KEY"},
					},
				},
				Action: func(c *cli.Context) error {
					grpcPort := c.Int("grpc-port")
					httpPort := c.Int("http-port")
					kafkaBroker := c.String("kafka-broker")
					kafkaTopic := c.String("kafka-topic")
					databaseURL := c.String("database-url")
					jwtSecretKey := c.String("jwt-secret-key")

					// Set JWT secret key in the auth package
					auth.SetJWTSecret(jwtSecretKey)
					if jwtSecretKey == "" || jwtSecretKey == "your-very-secret-key-that-should-be-in-env" {
						log.Println("WARNING: JWT_SECRET_KEY is not set or using default. Please set a strong, unique key in production.")
					}

					// Initialize Database
					db, err := database.NewPostgresDB(databaseURL)
					if err != nil {
						return fmt.Errorf("failed to initialize database: %w", err)
					}
					defer db.Close()

					// Initialize DB schema
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					if err := db.InitSchema(ctx); err != nil {
						return fmt.Errorf("failed to initialize database schema: %w", err)
					}

					// Initialize Kafka Producer
					kafkaProducer, err := kafka.NewKafkaProducer(kafkaBroker, kafkaTopic)
					if err != nil {
						return fmt.Errorf("failed to initialize Kafka producer: %w", err)
					}
					defer kafkaProducer.Close()

					workerRepo := worker_repo.NewInMemoryWorkerRepository()
					var testRepo domain.TestRepository = db
					var testResultRepo domain.TestResultRepository = db
					var aggregatedResultRepo domain.AggregatedResultRepository = db

					masterUC := masterUsecase.NewMasterUsecase(workerRepo, testRepo, testResultRepo, aggregatedResultRepo, kafkaProducer)

					// Start HTTP Server for UI and API
					httpHandler := masterHTTP.NewHTTPHandler(masterUC, jwtSecretKey)
					httpServer := &http.Server{
						Addr:    fmt.Sprintf(":%d", httpPort),
						Handler: httpHandler.Router,
					}

					go func() {
						log.Printf("Master HTTP server listening on port %d", httpPort)
						if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
							log.Fatalf("Master HTTP server failed: %v", err)
						}
					}()

					// Start gRPC Server
					recoveryOpts := []recovery.Option{
						recovery.WithRecoveryHandlerContext(func(ctx context.Context, p interface{}) (err error) {
							log.Printf("Recovered from panic: %v", p)
							return status.Errorf(codes.Internal, "panic: %v", p)
						}),
					}
					grpcServer := grpc.NewServer(
						grpc.UnaryInterceptor(recovery.UnaryServerInterceptor(recoveryOpts...)),
						grpc.StreamInterceptor(recovery.StreamServerInterceptor(recoveryOpts...)),
					)
					masterGRPCServer := masterGRPC.NewGRPCServer(masterUC)
					pb.RegisterWorkerServiceServer(grpcServer, masterGRPCServer)
					pb.RegisterMasterServiceServer(grpcServer, masterGRPCServer)
					reflection.Register(grpcServer)

					grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
					if err != nil {
						return fmt.Errorf("failed to listen for Master gRPC: %w", err)
					}

					go func() {
						log.Printf("Master gRPC server listening on port %d", grpcPort)
						if err := grpcServer.Serve(grpcLis); err != nil && err != grpc.ErrServerStopped {
							log.Fatalf("Master gRPC server failed: %v", err)
						}
					}()

					// Graceful shutdown
					quit := make(chan os.Signal, 1)
					signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
					<-quit
					log.Println("Shutting down Master servers...")

					// Shut down HTTP server
					httpCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer httpCancel()
					if err := httpServer.Shutdown(httpCtx); err != nil {
						log.Printf("Master HTTP server shutdown error: %v", err)
					}

					// Shut down gRPC server
					grpcServer.GracefulStop()

					log.Println("Master servers gracefully stopped.")
					return nil
				},
			},
			{
				Name:  "worker",
				Usage: "Starts a Worker service",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "grpc-port",
						Aliases: []string{"gp"},
						Value:   50052,
						Usage:   "gRPC port for Worker service to receive assignments",
						EnvVars: []string{"GRPC_PORT"},
					},
					&cli.StringFlag{
						Name:    "master-address",
						Aliases: []string{"ma"},
						Value:   "localhost:50051",
						Usage:   "Master service gRPC address (host:port)",
						EnvVars: []string{"MASTER_ADDRESS"},
					},
					&cli.StringFlag{
						Name:    "kafka-broker",
						Aliases: []string{"kb"},
						Value:   "localhost:9092",
						Usage:   "Kafka broker address for producing results",
						EnvVars: []string{"KAFKA_BROKER"},
					},
					&cli.StringFlag{
						Name:    "kafka-topic",
						Aliases: []string{"kt"},
						Value:   "test_results",
						Usage:   "Kafka topic to produce test results to",
						EnvVars: []string{"KAFKA_TOPIC"},
					},
					&cli.StringFlag{
						Name:    "worker-id",
						Aliases: []string{"id"},
						Value:   "worker-1",
						Usage:   "Unique ID for this worker instance",
						EnvVars: []string{"WORKER_ID"},
					},
				},
				Action: func(c *cli.Context) error {
					workerGRPCPort := c.Int("grpc-port")
					masterAddress := c.String("master-address")
					kafkaBroker := c.String("kafka-broker")
					kafkaTopic := c.String("kafka-topic")
					workerID := c.String("worker-id")

					// Initialize Kafka Producer
					kafkaProducer, err := kafka.NewKafkaProducer(kafkaBroker, kafkaTopic)
					if err != nil {
						return fmt.Errorf("failed to initialize Kafka producer: %w", err)
					}
					defer kafkaProducer.Close()

					// Initialize Vegeta Adapter
					vegetaExecutor := vegeta.NewVegetaAdapter()

					// Connect to Master gRPC
					masterConn, err := grpc.Dial(masterAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						return fmt.Errorf("failed to connect to master gRPC server %s: %w", masterAddress, err)
					}
					defer masterConn.Close()

					masterClient := pb.NewWorkerServiceClient(masterConn)

					workerUC := workerUsecase.NewWorkerUsecase(workerID, masterClient, vegetaExecutor, kafkaProducer)

					// Start worker lifecycle (registration and status streaming)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					go func() {
						log.Printf("Worker %s starting lifecycle...", workerID)
						err := workerUC.StartWorkerLifecycle(ctx, workerGRPCPort) // Pass worker's gRPC port
						if err != nil {
							log.Fatalf("Worker lifecycle failed: %v", err)
						}
					}()

					// Start gRPC server for receiving test assignments (Master calls this)
					grpcServer := grpc.NewServer()
					pb.RegisterWorkerServiceServer(grpcServer, workerGRPC.NewGRPCServer(workerUC))

					lis, err := net.Listen("tcp", fmt.Sprintf(":%d", workerGRPCPort))
					if err != nil {
						return fmt.Errorf("failed to listen for Worker gRPC: %w", err)
					}

					go func() {
						log.Printf("Worker gRPC server listening on port %d", workerGRPCPort)
						if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
							log.Fatalf("Worker gRPC server failed: %v", err)
						}
					}()

					// Graceful shutdown
					quit := make(chan os.Signal, 1)
					signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
					<-quit
					log.Println("Shutting down Worker...")

					cancel() // Cancel context to stop worker lifecycle goroutine
					grpcServer.GracefulStop()

					log.Println("Worker gracefully stopped.")
					return nil
				},
			},
			{
				Name:  "consumer",
				Usage: "Starts the Kafka Consumer service",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "kafka-broker",
						Aliases: []string{"kb"},
						Value:   "localhost:9092",
						Usage:   "Kafka broker address",
						EnvVars: []string{"KAFKA_BROKER"},
					},
					&cli.StringFlag{
						Name:    "kafka-topic",
						Aliases: []string{"kt"},
						Value:   "test_results",
						Usage:   "Kafka topic to consume results from",
						EnvVars: []string{"KAFKA_TOPIC"},
					},
					&cli.StringFlag{
						Name:    "kafka-group",
						Aliases: []string{"kg"},
						Value:   "load_tester_consumer_group",
						Usage:   "Kafka consumer group ID",
						EnvVars: []string{"KAFKA_GROUP"},
					},
					&cli.StringFlag{
						Name:    "database-url",
						Aliases: []string{"db"},
						Value:   "postgres://user:password@localhost:5432/distributed_load_tester?sslmode=disable",
						Usage:   "PostgreSQL database connection URL",
						EnvVars: []string{"DATABASE_URL"},
					},
				},
				Action: func(c *cli.Context) error {
					kafkaBroker := c.String("kafka-broker")
					kafkaTopic := c.String("kafka-topic")
					kafkaGroup := c.String("kafka-group")
					databaseURL := c.String("database-url")

					// Initialize Database
					db, err := database.NewPostgresDB(databaseURL)
					if err != nil {
						return fmt.Errorf("failed to initialize database: %w", err)
					}
					defer db.Close()

					// Initialize DB schema
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					if err := db.InitSchema(ctx); err != nil {
						return fmt.Errorf("failed to initialize database schema: %w", err)
					}

					// Initialize Kafka Consumer
					kafkaConsumer, err := kafka.NewKafkaConsumer(kafkaBroker, kafkaTopic, kafkaGroup)
					if err != nil {
						return fmt.Errorf("failed to initialize Kafka consumer: %w", err)
					}
					defer kafkaConsumer.Close()

					consumerUC := consumerUsecase.NewConsumerUsecase(db, db, kafkaConsumer)

					ctx, cancel = context.WithCancel(context.Background())
					defer cancel()

					// Start consuming messages in a goroutine
					go func() {
						log.Printf("Consumer starting Kafka consumer for topic '%s'...", kafkaTopic)
						err := consumerUC.StartConsuming(ctx, kafkaTopic)
						if err != nil {
							log.Fatalf("Consumer Kafka consumer failed: %v", err)
						}
					}()

					// Graceful shutdown
					quit := make(chan os.Signal, 1)
					signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
					<-quit
					log.Println("Shutting down Consumer...")

					cancel() // Cancel context to stop consumer goroutine

					log.Println("Consumer gracefully stopped.")
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// getWorkerAddress attempts to get the worker's reachable IP address and port.
// This function is moved here as it's common helper logic now.
func getWorkerAddress(workerGRPCPort int) string {
	conn, err := net.Dial("udp", "8.8.8.8:80") // Connect to a public DNS server to get local IP
	if err != nil {
		log.Printf("Warning: Could not determine local IP: %v. Using localhost.", err)
		return fmt.Sprintf("localhost:%d", workerGRPCPort) // Default worker gRPC port
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return fmt.Sprintf("%s:%d", localAddr.IP.String(), workerGRPCPort) // Use worker's gRPC port
}

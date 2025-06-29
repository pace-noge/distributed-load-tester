package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/database"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/vegeta"
	workerGRPC "github.com/pace-noge/distributed-load-tester/internal/worker/delivery/grpc"
	workerUsecase "github.com/pace-noge/distributed-load-tester/internal/worker/usecase"
	pb "github.com/pace-noge/distributed-load-tester/proto"
)

// NewWorkerCommand creates the worker command
func NewWorkerCommand() *cli.Command {
	return &cli.Command{
		Name:  "worker",
		Usage: "Starts the Worker service",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "grpc-port",
				Aliases: []string{"gp"},
				Value:   50052,
				Usage:   "gRPC port for Worker service",
				EnvVars: []string{"WORKER_GRPC_PORT"},
			},
			&cli.StringFlag{
				Name:    "master-address",
				Aliases: []string{"ma"},
				Value:   "localhost:50051",
				Usage:   "Master service gRPC address (host:port)",
				EnvVars: []string{"MASTER_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "worker-id",
				Aliases: []string{"id"},
				Value:   "worker-1",
				Usage:   "Unique ID for this worker instance",
				EnvVars: []string{"WORKER_ID"},
			},
			&cli.StringFlag{
				Name:    "database-url",
				Aliases: []string{"db"},
				Value:   "postgres://postgres:postgres@localhost:5432/load_tester?sslmode=disable",
				Usage:   "PostgreSQL database connection URL",
				EnvVars: []string{"DATABASE_URL"},
			},
		},
		Action: runWorker,
	}
}

func runWorker(c *cli.Context) error {
	workerGRPCPort := c.Int("grpc-port")
	masterAddress := c.String("master-address")
	workerID := c.String("worker-id")
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

	// Initialize Vegeta Adapter
	vegetaExecutor := vegeta.NewVegetaAdapter()

	// Connect to Master gRPC
	masterConn, err := grpc.Dial(masterAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to master gRPC server %s: %w", masterAddress, err)
	}
	defer masterConn.Close()

	masterClient := pb.NewWorkerServiceClient(masterConn)

	// Create worker usecase with database access
	var testResultRepo domain.TestResultRepository = db
	workerUC := workerUsecase.NewWorkerUsecase(workerID, vegetaExecutor, masterClient, testResultRepo)

	// Start worker lifecycle (registration and status streaming)
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Printf("Worker %s starting lifecycle...", workerID)
		err := workerUC.StartWorkerLifecycle(ctx, workerGRPCPort)
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
		log.Printf("Worker gRPC server starting on port %d...", workerGRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
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
}

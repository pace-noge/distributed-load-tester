package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/vegeta"
	"github.com/pace-noge/distributed-load-tester/internal/utils"
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
				Value:   "", // Empty value will trigger auto-generation
				Usage:   "Unique ID for this worker instance (leave empty for auto-generated memorable name)",
				EnvVars: []string{"WORKER_ID"},
			},
		},
		Action: runWorker,
	}
}

func runWorker(c *cli.Context) error {
	workerGRPCPort := c.Int("grpc-port")
	masterAddress := c.String("master-address")
	workerID := c.String("worker-id")

	// Generate memorable worker name if not provided or generic
	if workerID == "" || workerID == "worker-1" || workerID == "worker-2" {
		workerID = utils.GenerateWorkerName()
		log.Printf("🎯 Generated memorable worker name: %s (Display: %s)", workerID, utils.GetWorkerDisplayName(workerID))
	} else {
		log.Printf("🔧 Using provided worker ID: %s", workerID)
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

	// Create worker usecase without database dependency
	workerUC := workerUsecase.NewWorkerUsecase(workerID, vegetaExecutor, masterClient)

	// Start worker lifecycle (registration and status streaming)
	ctx, cancel := context.WithCancel(context.Background())
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

package cmd

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
	"google.golang.org/grpc/reflection"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/auth"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/database"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/worker_repo"
	masterGRPC "github.com/pace-noge/distributed-load-tester/internal/master/delivery/grpc"
	masterHTTP "github.com/pace-noge/distributed-load-tester/internal/master/delivery/http"
	masterWebSocket "github.com/pace-noge/distributed-load-tester/internal/master/delivery/websocket"
	masterUsecase "github.com/pace-noge/distributed-load-tester/internal/master/usecase"
	pb "github.com/pace-noge/distributed-load-tester/proto"
)

// NewMasterCommand creates the master command
func NewMasterCommand() *cli.Command {
	return &cli.Command{
		Name:  "master",
		Usage: "Starts the Master service",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "grpc-port",
				Aliases: []string{"gp"},
				Value:   50051,
				Usage:   "gRPC port for Master service",
				EnvVars: []string{"MASTER_GRPC_PORT"},
			},
			&cli.IntFlag{
				Name:    "http-port",
				Aliases: []string{"hp"},
				Value:   8080,
				Usage:   "HTTP port for Master service",
				EnvVars: []string{"MASTER_HTTP_PORT"},
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
		Action: runMaster,
	}
}

func runMaster(c *cli.Context) error {
	grpcPort := c.Int("grpc-port")
	httpPort := c.Int("http-port")
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

	workerRepo := worker_repo.NewInMemoryWorkerRepository()
	var testRepo domain.TestRepository = db
	var testResultRepo domain.TestResultRepository = db
	var aggregatedResultRepo domain.AggregatedResultRepository = db

	masterUC := masterUsecase.NewMasterUsecase(workerRepo, testRepo, testResultRepo, aggregatedResultRepo)

	// Start aggregation background job
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	go masterUC.StartAggregationBackgroundJob(bgCtx, 2*time.Minute) // Check every 2 minutes
	log.Println("Started aggregation background job")

	// Initialize WebSocket handler
	wsHandler := masterWebSocket.NewWebSocketHandler(masterUC, jwtSecretKey)
	go wsHandler.StartHub(bgCtx)

	// Initialize HTTP handler
	httpHandler := masterHTTP.NewHTTPHandler(masterUC, jwtSecretKey)

	// Register WebSocket handler with HTTP handler
	httpHandler.RegisterWebSocketHandler(wsHandler.HandleWebSocket)

	// Start gRPC server
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(recovery.UnaryServerInterceptor()))
	masterGRPCHandler := masterGRPC.NewGRPCServer(masterUC)

	// Register both services on the same server
	pb.RegisterMasterServiceServer(grpcServer, masterGRPCHandler)
	pb.RegisterWorkerServiceServer(grpcServer, masterGRPCHandler)
	reflection.Register(grpcServer)

	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		return fmt.Errorf("failed to listen for Master gRPC: %w", err)
	}

	go func() {
		log.Printf("Master gRPC server starting on port %d...", grpcPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("Master gRPC server failed: %v", err)
		}
	}()

	// Start HTTP server
	go func() {
		log.Printf("Master HTTP server starting on port %d...", httpPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), httpHandler.Router); err != nil {
			log.Fatalf("Master HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Master...")

	bgCancel() // Cancel background jobs
	grpcServer.GracefulStop()

	log.Println("Master gracefully stopped.")
	return nil
}

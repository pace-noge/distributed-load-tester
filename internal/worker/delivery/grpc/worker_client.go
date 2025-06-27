// internal/worker/delivery/grpc/worker_client.go
package grpc

import (
	"context"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	workerUsecase "github.com/pace-noge/distributed-load-tester/internal/worker/usecase"
	pb "github.com/pace-noge/distributed-load-tester/proto"
)

// GRPCServer implements the gRPC WorkerServiceServer interface for a worker.
// This server receives test assignments from the Master.
type GRPCServer struct {
	pb.UnimplementedWorkerServiceServer
	usecase *workerUsecase.WorkerUsecase
}

// NewGRPCServer creates a new GRPCServer instance for the worker.
func NewGRPCServer(uc *workerUsecase.WorkerUsecase) *GRPCServer {
	return &GRPCServer{
		usecase: uc,
	}
}

// AssignTest receives a test assignment from the Master and triggers the execution.
func (s *GRPCServer) AssignTest(ctx context.Context, req *pb.TestAssignment) (*pb.AssignmentResponse, error) {
	log.Printf("Worker received test assignment for Test ID: %s", req.TestId)

	testAssignment := &domain.TestAssignment{
		TestID:            req.TestId,
		VegetaPayloadJSON: req.VegetaPayloadJson,
		DurationSeconds:   req.DurationSeconds,
		RatePerSecond:     req.RatePerSecond,
		TargetsBase64:     req.TargetsBase64,
	}

	// Execute test asynchronously to avoid blocking the assignment RPC
	go func() {
		err := s.usecase.ExecuteTest(context.Background(), testAssignment)
		if err != nil {
			log.Printf("Worker failed to execute test %s: %v", req.TestId, err)
		}
	}()

	return &pb.AssignmentResponse{Accepted: true, Message: "Test assignment accepted and execution started."}, nil
}

// RegisterWorker is not implemented on the worker's gRPC server, only on master.
func (s *GRPCServer) RegisterWorker(ctx context.Context, req *pb.WorkerInfo) (*pb.RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterWorker not implemented by worker")
}

// StreamWorkerStatus is not implemented on the worker's gRPC server.
// The worker is the *client* for this bidirectional stream, sending status to the master.
// However, to satisfy the pb.WorkerServiceServer interface (which the worker's gRPC server implements
// to receive AssignTest calls), this method must have the correct signature for a bidirectional stream.
func (s *GRPCServer) StreamWorkerStatus(stream pb.WorkerService_StreamWorkerStatusServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamWorkerStatus not implemented by worker's server (worker is a client for this RPC, not a server)")
}

// internal/master/delivery/grpc/master_server.go
package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	masterUsecase "github.com/pace-noge/distributed-load-tester/internal/master/usecase"
	pb "github.com/pace-noge/distributed-load-tester/proto"
)

// GRPCServer implements the gRPC WorkerServiceServer and MasterServiceServer interfaces.
type GRPCServer struct {
	pb.UnimplementedWorkerServiceServer
	pb.UnimplementedMasterServiceServer
	usecase *masterUsecase.MasterUsecase
}

// NewGRPCServer creates a new GRPCServer instance.
func NewGRPCServer(uc *masterUsecase.MasterUsecase) *GRPCServer {
	return &GRPCServer{
		usecase: uc,
	}
}

// RegisterWorker handles worker registration (Unary RPC).
func (s *GRPCServer) RegisterWorker(ctx context.Context, req *pb.WorkerInfo) (*pb.RegisterResponse, error) {
	log.Printf("Worker %s attempting to register from %s", req.Id, req.Address)
	worker := &domain.Worker{
		ID:       req.Id,
		Address:  req.Address,
		Status:   "READY", // Initial status
		LastSeen: time.Now(),
	}
	err := s.usecase.RegisterWorker(ctx, worker)
	if err != nil {
		log.Printf("Failed to register worker %s: %v", req.Id, err)
		return &pb.RegisterResponse{Success: false, Message: fmt.Sprintf("Failed to register: %v", err)}, status.Errorf(codes.Internal, "registration failed: %v", err)
	}
	log.Printf("Worker %s registered successfully.", req.Id)
	return &pb.RegisterResponse{Success: true, Message: "Worker registered successfully"}, nil
}

// StreamWorkerStatus handles bidirectional streaming for worker status updates.
// The Master receives WorkerStatus messages and can send WorkerStatusAck messages.
func (s *GRPCServer) StreamWorkerStatus(stream pb.WorkerService_StreamWorkerStatusServer) error {
	var workerID string // To be set from the first message
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			if workerID != "" {
				log.Printf("Worker %s stream disconnected (context done). Marking offline.", workerID)
				markOfflineCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				s.usecase.MarkWorkerOffline(markOfflineCtx, workerID)
			}
			return ctx.Err()
		default:
			statusMsg, err := stream.Recv()
			if err == io.EOF {
				if workerID != "" {
					log.Printf("Worker %s stream closed by client. Marking offline.", workerID)
					markOfflineCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					s.usecase.MarkWorkerOffline(markOfflineCtx, workerID)
				}
				return nil
			}
			if err != nil {
				log.Printf("Error receiving worker status from %s: %v", workerID, err)
				if workerID != "" {
					markOfflineCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					s.usecase.MarkWorkerOffline(markOfflineCtx, workerID)
				}
				return status.Errorf(codes.Unavailable, "stream error: %v", err)
			}

			// Set workerID from the first message if not already set
			if workerID == "" {
				workerID = statusMsg.WorkerId
				log.Printf("First status received from worker: %s. Starting status stream handling.", workerID)
			} else if statusMsg.WorkerId != workerID {
				// Prevent worker impersonation or mixed streams
				log.Printf("Mismatched worker ID in stream: expected %s, got %s. Closing stream.", workerID, statusMsg.WorkerId)
				return status.Errorf(codes.InvalidArgument, "worker ID mismatch in stream")
			}

			log.Printf("Received status from worker %s: %s, test: %s, progress: %d/%d",
				statusMsg.WorkerId, statusMsg.Status.String(), statusMsg.TestId, statusMsg.CompletedRequests, statusMsg.TotalRequests)

			// Update worker status in usecase
			err = s.usecase.UpdateWorkerStatus(ctx, statusMsg.WorkerId, statusMsg.Status.String(), statusMsg.TestId,
				statusMsg.Message, statusMsg.CompletedRequests, statusMsg.TotalRequests)
			if err != nil {
				log.Printf("Error updating worker status for %s: %v", statusMsg.WorkerId, err)
				// Send a negative ACK back if status update fails
				stream.Send(&pb.WorkerStatusAck{Accepted: false, Message: fmt.Sprintf("Failed to update status: %v", err)})
			} else {
				// Send a positive ACK back
				stream.Send(&pb.WorkerStatusAck{Accepted: true, Message: "Status received"})
			}

			// If worker signals completion/error for a test, update test status
			if statusMsg.TestId != "" && (statusMsg.Status == pb.StatusType_FINISHING || statusMsg.Status == pb.StatusType_ERROR) {
				log.Printf("Worker %s signaling test %s completion/error.", statusMsg.WorkerId, statusMsg.TestId)
				s.usecase.HandleWorkerTestCompletion(ctx, statusMsg.TestId, statusMsg.WorkerId, statusMsg.Status == pb.StatusType_ERROR)
			}
		}
	}
}

// AssignTest handles test assignment from Master to Worker (Unary RPC).
func (s *GRPCServer) AssignTest(ctx context.Context, req *pb.TestAssignment) (*pb.AssignmentResponse, error) {
	// This method is called by the MasterUsecase to assign a test to a specific worker.
	log.Printf("Received direct test assignment request for test %s (internal call, should not be direct from worker)", req.TestId)
	return &pb.AssignmentResponse{Accepted: true, Message: "Assignment acknowledged (internal)."}, nil
}

// SubmitTest handles external API requests to submit a new test (Unary RPC).
func (s *GRPCServer) SubmitTest(ctx context.Context, req *pb.TestRequest) (*pb.TestSubmissionResponse, error) {
	// Authentication/Authorization check (basic example)
	if req.RequesterId == "" {
		return &pb.TestSubmissionResponse{Success: false, Message: "Unauthorized: Requester ID missing"}, status.Errorf(codes.Unauthenticated, "requester ID missing")
	}

	testReq := &domain.TestRequest{
		Name:              req.Name,
		VegetaPayloadJSON: req.VegetaPayloadJson,
		DurationSeconds:   req.DurationSeconds,
		RatePerSecond:     req.RatePerSecond,
		TargetsBase64:     req.TargetsBase64,
		RequesterID:       req.RequesterId,
	}

	testID, err := s.usecase.SubmitTest(ctx, testReq)
	if err != nil {
		log.Printf("Error submitting test: %v", err)
		return &pb.TestSubmissionResponse{Success: false, Message: fmt.Sprintf("Test submission failed: %v", err)}, status.Errorf(codes.Internal, "test submission failed: %v", err)
	}

	log.Printf("Test submitted successfully with ID: %s", testID)
	return &pb.TestSubmissionResponse{TestId: testID, Success: true, Message: "Test submitted successfully"}, nil
}

// GetDashboardStatus provides dashboard data for the UI (Unary RPC).
func (s *GRPCServer) GetDashboardStatus(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardStatus, error) {
	dashboard, err := s.usecase.GetDashboardStatus(ctx)
	if err != nil {
		log.Printf("Error getting dashboard status: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get dashboard status: %v", err)
	}

	pbActiveTests := make([]*pb.ActiveTest, len(dashboard.ActiveTests))
	for i, at := range dashboard.ActiveTests {
		pbActiveTests[i] = &pb.ActiveTest{
			TestId:                 at.TestID,
			TestName:               at.TestName,
			AssignedWorkers:        at.AssignedWorkers,
			CompletedWorkers:       at.CompletedWorkers,
			FailedWorkers:          at.FailedWorkers,
			Status:                 at.Status,
			TotalRequestsSent:      at.TotalRequestsSent,
			TotalRequestsCompleted: at.TotalRequestsCompleted,
			TotalDurationMs:        at.TotalDurationMs,
		}
	}

	pbWorkerSummaries := make([]*pb.WorkerSummary, len(dashboard.WorkerSummaries))
	for i, ws := range dashboard.WorkerSummaries {
		statusType := pb.StatusType_READY // Default
		switch ws.StatusType {
		case "READY":
			statusType = pb.StatusType_READY
		case "BUSY":
			statusType = pb.StatusType_BUSY
		case "FINISHING":
			statusType = pb.StatusType_FINISHING
		case "ERROR":
			statusType = pb.StatusType_ERROR
		}

		pbWorkerSummaries[i] = &pb.WorkerSummary{
			WorkerId:          ws.WorkerID,
			StatusMessage:     ws.StatusMessage,
			StatusType:        statusType,
			CurrentTestId:     ws.CurrentTestID,
			CompletedRequests: ws.CompletedRequests,
			TotalRequests:     ws.TotalRequests,
		}
	}

	return &pb.DashboardStatus{
		TotalWorkers:     dashboard.TotalWorkers,
		AvailableWorkers: dashboard.AvailableWorkers,
		BusyWorkers:      dashboard.BusyWorkers,
		ActiveTests:      pbActiveTests,
		WorkerSummaries:  pbWorkerSummaries,
	}, nil
}

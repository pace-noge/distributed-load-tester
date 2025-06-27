// internal/worker/usecase/worker_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io" // For io.EOF
	"log"
	"net"
	"sync" // For sync.Once and mutex
	"time"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	pb "github.com/pace-noge/distributed-load-tester/proto"
)

// WorkerUsecase handles the business logic for the worker service.
type WorkerUsecase struct {
	workerID       string
	masterClient   pb.WorkerServiceClient
	vegetaExecutor domain.VegetaExecutor
	kafkaProducer  domain.KafkaProducer
	currentTestID  string // Tracks the ID of the test currently being executed

	statusStreamClient pb.WorkerService_StreamWorkerStatusClient
	statusStreamCancel context.CancelFunc // To cancel the status stream context
	statusStreamOnce   sync.Once          // Ensures stream is established only once
	statusStreamMu     sync.Mutex         // Protects sending on the stream
}

// NewWorkerUsecase creates a new WorkerUsecase instance.
func NewWorkerUsecase(workerID string, masterClient pb.WorkerServiceClient, ve domain.VegetaExecutor, kp domain.KafkaProducer) *WorkerUsecase {
	return &WorkerUsecase{
		workerID:       workerID,
		masterClient:   masterClient,
		vegetaExecutor: ve,
		kafkaProducer:  kp,
	}
}

// StartWorkerLifecycle registers the worker with the master and starts the bidirectional status stream.
func (uc *WorkerUsecase) StartWorkerLifecycle(ctx context.Context, workerGRPCPort int) error {
	workerInfo := &pb.WorkerInfo{
		Id:      uc.workerID,
		Address: getWorkerAddress(workerGRPCPort),
	}
	log.Printf("Attempting to register worker %s with master at %s", uc.workerID, workerInfo.Address)

	// Step 1: Register with Master (Unary RPC)
	// This part is a simple unary RPC.
	var regResp *pb.RegisterResponse
	var regErr error
	for i := 0; i < 5; i++ { // Retry 5 times
		regResp, regErr = uc.masterClient.RegisterWorker(ctx, workerInfo)
		if regErr != nil {
			log.Printf("Attempt %d: Failed to register worker %s with master: %v. Retrying in 5s...", i+1, uc.workerID, regErr)
			time.Sleep(5 * time.Second)
			continue
		}
		if !regResp.Success {
			log.Printf("Attempt %d: Master rejected worker %s registration: %s. Retrying in 5s...", i+1, uc.workerID, regResp.Message)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf("Worker %s registered successfully with master.", uc.workerID)
		break
	}

	if regErr != nil || !regResp.Success {
		return fmt.Errorf("failed to register worker after multiple retries: %v", regErr)
	}

	// Step 2: Establish the bidirectional status stream
	uc.statusStreamOnce.Do(func() {
		streamCtx, streamCancel := context.WithCancel(context.Background())
		uc.statusStreamCancel = streamCancel

		stream, err := uc.masterClient.StreamWorkerStatus(streamCtx) // Initiate bidirectional stream
		if err != nil {
			log.Fatalf("Worker %s failed to open status stream to master: %v", uc.workerID, err)
		}
		uc.statusStreamClient = stream
		log.Printf("Worker %s established bidirectional status stream to master.", uc.workerID)

		// Goroutine to send periodic heartbeats/status updates
		go uc.sendPeriodicStatusUpdates(streamCtx)

		// Goroutine to receive acknowledgments/commands from master
		go uc.receiveStreamResponses(streamCtx)
	})

	// Send initial READY status through the newly established stream
	return uc.sendStatusToMaster(
		pb.StatusType_READY,
		"Worker initialized and ready",
		"", 0, 0, 0,
	)
}

// getWorkerAddress attempts to get the worker's reachable IP address and port.
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

// sendStatusToMaster sends a WorkerStatus message over the bidirectional stream.
// It tries to re-establish the stream if it's broken.
func (uc *WorkerUsecase) sendStatusToMaster(statusType pb.StatusType, message, testID string, totalReq, completedReq, durationMs int64) error {
	uc.statusStreamMu.Lock()
	defer uc.statusStreamMu.Unlock()

	statusMsg := &pb.WorkerStatus{
		WorkerId:          uc.workerID,
		Status:            statusType,
		Message:           message,
		TestId:            testID,
		TotalRequests:     totalReq,
		CompletedRequests: completedReq,
		DurationMs:        durationMs,
	}

	// Retry sending status in case of stream issues
	for i := 0; i < 3; i++ {
		if uc.statusStreamClient == nil {
			log.Printf("Status stream client is nil. Attempting to re-establish. Attempt %d...", i+1)
			uc.reestablishStatusStream()
			if uc.statusStreamClient == nil {
				time.Sleep(time.Second)
				continue
			}
		}

		err := uc.statusStreamClient.Send(statusMsg)
		if err == nil {
			log.Printf("Worker %s sent status: %s (Test: %s)", uc.workerID, statusType.String(), testID)
			return nil
		}

		log.Printf("Worker %s failed to send status update (attempt %d): %v. Re-establishing stream...", uc.workerID, i+1, err)
		uc.statusStreamClient.CloseSend() // Close the current broken stream
		uc.statusStreamClient = nil       // Mark for re-establishment
		time.Sleep(time.Second)           // Small backoff before retrying
		uc.reestablishStatusStream()      // Attempt to re-establish
	}
	return fmt.Errorf("failed to send status after multiple retries")
}

// reestablishStatusStream attempts to create a new bidirectional status stream.
func (uc *WorkerUsecase) reestablishStatusStream() {
	// This function should ideally be called under a lock if used concurrently,
	// but here it's called within `sendStatusToMaster` which already holds a lock.
	if uc.statusStreamClient != nil {
		return // Stream is already active or being re-established by another call
	}

	// Create a new context for the new stream
	streamCtx, streamCancel := context.WithCancel(context.Background())
	uc.statusStreamCancel = streamCancel // Update the cancel func

	newStream, err := uc.masterClient.StreamWorkerStatus(streamCtx)
	if err != nil {
		log.Printf("Worker %s failed to re-establish status stream: %v", uc.workerID, err)
		uc.statusStreamClient = nil // Ensure it remains nil on failure
		return
	}
	uc.statusStreamClient = newStream
	log.Printf("Worker %s successfully re-established status stream.", uc.workerID)

	// Restart receive goroutine for the new stream
	go uc.receiveStreamResponses(streamCtx)
}

// sendPeriodicStatusUpdates sends a "READY" heartbeat or current test progress periodically.
func (uc *WorkerUsecase) sendPeriodicStatusUpdates(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %s periodic status sender stopped.", uc.workerID)
			return
		case <-ticker.C:
			var statusType pb.StatusType
			var message string
			var testID string
			var completedReqs, totalReqs int64

			if uc.currentTestID != "" {
				statusType = pb.StatusType_BUSY
				message = fmt.Sprintf("Running test %s", uc.currentTestID)
				testID = uc.currentTestID
				// In a real scenario, you'd store and update actual progress during test execution.
				// For now, these are placeholders unless updated by ExecuteTest.
				// completedReqs = uc.currentTestProgress.completed
				// totalReqs = uc.currentTestProgress.total
			} else {
				statusType = pb.StatusType_READY
				message = "Ready for new tests"
			}

			err := uc.sendStatusToMaster(statusType, message, testID, totalReqs, completedReqs, 0)
			if err != nil {
				log.Printf("Worker %s failed to send periodic status: %v", uc.workerID, err)
				// Error handling for persistent stream failures would go here (e.g., exponential backoff)
			}
		}
	}
}

// receiveStreamResponses listens for messages from the Master on the bidirectional stream.
func (uc *WorkerUsecase) receiveStreamResponses(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %s stream receiver stopped.", uc.workerID)
			return
		default:
			if uc.statusStreamClient == nil {
				time.Sleep(time.Second) // Wait for stream to be re-established
				continue
			}
			ack, err := uc.statusStreamClient.Recv()
			if err == io.EOF {
				log.Printf("Master closed status stream to worker %s. Attempting to re-establish.", uc.workerID)
				uc.statusStreamClient = nil // Mark for re-establishment
				uc.reestablishStatusStream()
				time.Sleep(time.Second) // Small backoff
				continue
			}
			if err != nil {
				log.Printf("Error receiving from master on status stream for worker %s: %v. Attempting to re-establish.", uc.workerID, err)
				uc.statusStreamClient = nil // Mark for re-establishment
				uc.reestablishStatusStream()
				time.Sleep(time.Second) // Small backoff
				continue
			}
			log.Printf("Received ACK from Master for worker %s: Success=%t, Message=%s", uc.workerID, ack.Accepted, ack.Message)
			// Handle any specific commands/acks from master here
		}
	}
}

// ExecuteTest takes a test assignment and runs the Vegeta load test.
func (uc *WorkerUsecase) ExecuteTest(ctx context.Context, assignment *domain.TestAssignment) error {
	uc.currentTestID = assignment.TestID // Set current test ID

	log.Printf("Worker %s starting test %s...", uc.workerID, assignment.TestID)

	// Inform master that worker is busy
	err := uc.sendStatusToMaster(
		pb.StatusType_BUSY,
		fmt.Sprintf("Running test %s", assignment.TestID),
		assignment.TestID, 0, 0, 0,
	)
	if err != nil {
		log.Printf("Warning: Failed to send busy status to master for test %s: %v", assignment.TestID, err)
		// Proceed with test, but master might not know worker is busy
	}

	// Execute Vegeta attack
	result, err := uc.vegetaExecutor.Attack(ctx, assignment.VegetaPayloadJSON, assignment.DurationSeconds, assignment.RatePerSecond, assignment.TargetsBase64)
	if err != nil {
		log.Printf("Worker %s failed to execute Vegeta attack for test %s: %v", uc.workerID, assignment.TestID, err)
		// Send ERROR status to master
		sendErr := uc.sendStatusToMaster(
			pb.StatusType_ERROR,
			fmt.Sprintf("Test failed: %v", err),
			assignment.TestID, 0, 0, 0,
		)
		if sendErr != nil {
			log.Printf("Warning: Could not send error status to master: %v", sendErr)
		}
		uc.currentTestID = "" // Clear current test
		return fmt.Errorf("vegeta attack failed: %w", err)
	}

	result.TestID = assignment.TestID
	result.WorkerID = uc.workerID

	// Produce result to Kafka
	resultBytes, err := json.Marshal(result) // Marshal the domain.TestResult
	if err != nil {
		log.Printf("Failed to marshal test result for Kafka: %v", err)
		// Still send FINISHING status even if Kafka fails
	} else {
		produceCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = uc.kafkaProducer.Produce(produceCtx, "test_results", assignment.TestID, resultBytes)
		if err != nil {
			log.Printf("Failed to produce test result to Kafka for test %s: %v", assignment.TestID, err)
			// Decide if this should lead to a test failure or just a warning
		} else {
			log.Printf("Worker %s successfully produced result to Kafka for test %s", uc.workerID, assignment.TestID)
		}
	}

	// Inform master that test is finished
	sendErr := uc.sendStatusToMaster(
		pb.StatusType_FINISHING,
		"Test completed and results sent",
		assignment.TestID, result.TotalRequests, result.CompletedRequests, result.DurationMs,
	)
	if sendErr != nil {
		log.Printf("Warning: Could not send finishing status to master: %v", sendErr)
	}

	log.Printf("Worker %s finished test %s.", uc.workerID, assignment.TestID)
	uc.currentTestID = "" // Clear current test

	return nil
}

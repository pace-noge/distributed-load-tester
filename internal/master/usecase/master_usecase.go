package usecase

import (
	"context"
	"fmt"
	"log"
	"sort" // Required for sorting p95Latencies
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	pb "github.com/pace-noge/distributed-load-tester/proto"
)

// MasterUsecase handles the business logic for the master service.
type MasterUsecase struct {
	workerRepo            domain.WorkerRepository
	testRepo              domain.TestRepository
	testResultRepo        domain.TestResultRepository
	aggregatedResultRepo  domain.AggregatedResultRepository
	activeWorkerClients   sync.Map // Map[string]*grpc.ClientConn
	activeTestAssignments sync.Map // Map[string]map[string]bool // testID -> workerID -> assigned
	// For managing test distribution to workers
	testQueue          chan *domain.TestRequest
	workerAvailability chan string     // Channel for available worker IDs
	availableWorkers   map[string]bool // Track which workers are already in the availability queue
	mu                 sync.Mutex      // Protects access to testQueue, workerAvailability, and availableWorkers
}

// NewMasterUsecase creates a new MasterUsecase instance.
func NewMasterUsecase(
	wr domain.WorkerRepository,
	tr domain.TestRepository,
	trr domain.TestResultRepository,
	arr domain.AggregatedResultRepository) *MasterUsecase {

	uc := &MasterUsecase{
		workerRepo:           wr,
		testRepo:             tr,
		testResultRepo:       trr,
		aggregatedResultRepo: arr,
		testQueue:            make(chan *domain.TestRequest, 100), // Buffered channel for tests
		workerAvailability:   make(chan string, 200),              // Buffered channel for available worker IDs
		availableWorkers:     make(map[string]bool),               // Track workers in availability queue
	}
	go uc.startTestDistributionRoutine()
	return uc
}

// RegisterWorker registers a new worker with the master.
func (uc *MasterUsecase) RegisterWorker(ctx context.Context, worker *domain.Worker) error {
	// Attempt to connect to the worker's gRPC endpoint
	conn, err := grpc.DialContext(ctx, worker.Address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to connect to worker %s at %s: %w", worker.ID, worker.Address, err)
	}

	uc.activeWorkerClients.Store(worker.ID, conn)
	worker.Status = "READY"
	err = uc.workerRepo.RegisterWorker(ctx, worker)
	if err != nil {
		conn.Close() // Close connection if DB registration fails
		uc.activeWorkerClients.Delete(worker.ID)
		return fmt.Errorf("failed to save worker to repository: %w", err)
	}

	// Add worker to availability queue
	uc.addWorkerToAvailabilityQueue(worker.ID)
	return nil
}

// UpdateWorkerStatus updates the status of a worker.
func (uc *MasterUsecase) UpdateWorkerStatus(ctx context.Context, workerID string, status string, currentTestID string, progressMsg string, completedReqs, totalReqs int64) error {
	err := uc.workerRepo.UpdateWorkerStatus(ctx, workerID, status, currentTestID, progressMsg, completedReqs, totalReqs)
	if err != nil {
		log.Printf("Error updating worker status in repo for %s: %v", workerID, err)
		return err
	}

	// If worker becomes READY, push to availability queue
	if status == "READY" {
		uc.addWorkerToAvailabilityQueue(workerID)
	}
	return nil
}

// MarkWorkerOffline marks a worker as offline.
func (uc *MasterUsecase) MarkWorkerOffline(ctx context.Context, workerID string) error {
	log.Printf("Marking worker %s offline...", workerID)
	err := uc.workerRepo.MarkWorkerOffline(ctx, workerID)
	if err != nil {
		log.Printf("Failed to mark worker %s offline in DB: %v", workerID, err)
		// Don't return error to allow other cleanup
	}

	// Close gRPC connection and remove from active clients
	if connVal, ok := uc.activeWorkerClients.LoadAndDelete(workerID); ok {
		if conn, ok := connVal.(*grpc.ClientConn); ok {
			conn.Close()
			log.Printf("Closed gRPC connection for worker %s", workerID)
		}
	}
	return nil
}

// SubmitTest receives a test request and puts it in a queue for assignment.
func (uc *MasterUsecase) SubmitTest(ctx context.Context, testReq *domain.TestRequest) (string, error) {
	testReq.ID = uuid.New().String()
	testReq.CreatedAt = time.Now()
	testReq.Status = "PENDING"
	testReq.AssignedWorkersIDs = []string{}
	testReq.CompletedWorkers = []string{}
	testReq.FailedWorkers = []string{}

	// Set default worker count if not specified
	if testReq.WorkerCount == 0 {
		testReq.WorkerCount = 1
	}

	// Set default rate distribution mode if not specified
	if testReq.RateDistribution == "" {
		testReq.RateDistribution = "shared" // Default to shared distribution
	}

	// Validate rate distribution mode
	validModes := []string{"shared", "same", "weighted", "ramped", "burst"}
	isValid := false
	for _, mode := range validModes {
		if testReq.RateDistribution == mode {
			isValid = true
			break
		}
	}
	if !isValid {
		return "", fmt.Errorf("invalid rate_distribution: must be one of %v", validModes)
	}

	// Validate weighted distribution
	if testReq.RateDistribution == "weighted" {
		if len(testReq.RateWeights) == 0 {
			return "", fmt.Errorf("rate_weights must be provided for weighted distribution")
		}
		if len(testReq.RateWeights) != int(testReq.WorkerCount) {
			return "", fmt.Errorf("rate_weights length (%d) must match worker_count (%d)", len(testReq.RateWeights), testReq.WorkerCount)
		}
		// Validate weights are positive
		for i, weight := range testReq.RateWeights {
			if weight <= 0 {
				return "", fmt.Errorf("rate_weights[%d] must be positive, got %f", i, weight)
			}
		}
	}

	err := uc.testRepo.SaveTestRequest(ctx, testReq)
	if err != nil {
		return "", fmt.Errorf("failed to save test request: %w", err)
	}

	// Put test into queue for assignment
	select {
	case uc.testQueue <- testReq:
		log.Printf("Test %s submitted and added to assignment queue (requires %d workers, rate distribution: %s).",
			testReq.ID, testReq.WorkerCount, testReq.RateDistribution)
		return testReq.ID, nil
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Second): // Timeout if queue is full
		return "", fmt.Errorf("test queue is full, please try again later")
	}
}

// startTestDistributionRoutine is a goroutine that continuously assigns tests to available workers.
func (uc *MasterUsecase) startTestDistributionRoutine() {
	log.Println("Starting test distribution routine...")
	for {
		select {
		case testReq := <-uc.testQueue:
			log.Printf("Picked up test %s from queue. Looking for %d available workers...", testReq.ID, testReq.WorkerCount)

			// Collect the required number of workers
			var assignedWorkers []string
			timeout := time.After(30 * time.Second) // Wait up to 30 seconds to gather workers

			for uint32(len(assignedWorkers)) < testReq.WorkerCount {
				select {
				case workerID := <-uc.workerAvailability:
					assignedWorkers = append(assignedWorkers, workerID)
					uc.removeWorkerFromAvailabilityQueue(workerID) // Remove from tracking
					log.Printf("Worker %s assigned to test %s (%d/%d workers collected)",
						workerID, testReq.ID, len(assignedWorkers), testReq.WorkerCount)
				case <-timeout:
					log.Printf("Timeout waiting for workers for test %s. Only %d/%d workers available",
						testReq.ID, len(assignedWorkers), testReq.WorkerCount)

					// If we have at least one worker, proceed with partial assignment
					if len(assignedWorkers) > 0 {
						log.Printf("Proceeding with partial assignment for test %s using %d workers",
							testReq.ID, len(assignedWorkers))
						break
					} else {
						// No workers available, re-queue the test
						log.Printf("No workers available for test %s, re-queueing", testReq.ID)
						select {
						case uc.testQueue <- testReq:
						default:
							log.Printf("Failed to re-queue test %s, marking as failed", testReq.ID)
							uc.testRepo.UpdateTestStatus(context.Background(), testReq.ID, "FAILED",
								testReq.CompletedWorkers, append(testReq.FailedWorkers, "NoWorkersAvailable"))
						}
						continue
					}
				}
			}

			// Assign test to all collected workers concurrently
			uc.assignTestToMultipleWorkers(context.Background(), testReq, assignedWorkers)

		case <-time.After(10 * time.Second):
			// Periodically check for workers that might have gone offline without notifying
			// and re-queue tests if assigned to offline workers.
			uc.cleanupStaleWorkers(context.Background())
			// Also check for stuck tests due to worker count mismatches
			uc.fixStuckTests(context.Background())
		}
	}
}

// assignTestToWorker sends a test assignment to a specific worker via gRPC.
func (uc *MasterUsecase) assignTestToWorker(ctx context.Context, testReq *domain.TestRequest, workerID string) {
	connVal, ok := uc.activeWorkerClients.Load(workerID)
	if !ok {
		log.Printf("Worker %s connection not found. Re-queueing test %s.", workerID, testReq.ID)
		select {
		case uc.testQueue <- testReq: // Re-queue the test
		default:
			log.Printf("Failed to re-queue test %s, test queue full.", testReq.ID)
			// Mark test as failed if it can't be re-queued
			uc.testRepo.UpdateTestStatus(ctx, testReq.ID, "FAILED", testReq.CompletedWorkers, append(testReq.FailedWorkers, "NoWorkersAvailable"))
		}
		// Also mark worker as offline if it was expected to be available but isn't
		uc.MarkWorkerOffline(ctx, workerID)
		return
	}

	conn := connVal.(*grpc.ClientConn)
	client := pb.NewWorkerServiceClient(conn)

	// Update test status to RUNNING (but don't assign worker until successful)
	uc.testRepo.UpdateTestStatus(ctx, testReq.ID, "RUNNING", nil, nil) // Update overall test status

	// Mark worker as busy
	uc.workerRepo.UpdateWorkerStatus(ctx, workerID, "BUSY", testReq.ID, "Assigned test", 0, 0)

	assignment := &pb.TestAssignment{
		TestId:            testReq.ID,
		VegetaPayloadJson: testReq.VegetaPayloadJSON,
		DurationSeconds:   testReq.DurationSeconds,
		RatePerSecond:     testReq.RatePerSecond,
		TargetsBase64:     testReq.TargetsBase64,
	}

	assignmentCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Timeout for assignment RPC
	defer cancel()

	resp, err := client.AssignTest(assignmentCtx, assignment)
	if err != nil {
		log.Printf("Failed to assign test %s to worker %s: %v", testReq.ID, workerID, err)
		// Mark worker as offline, re-queue test
		uc.MarkWorkerOffline(ctx, workerID)
		uc.testRepo.AddFailedWorkerToTest(ctx, testReq.ID, workerID)
		select {
		case uc.testQueue <- testReq:
			log.Printf("Test %s re-queued due to assignment failure with worker %s.", testReq.ID, workerID)
		default:
			log.Printf("Failed to re-queue test %s, test queue full. Marking test as failed.", testReq.ID)
			uc.testRepo.UpdateTestStatus(ctx, testReq.ID, "FAILED", testReq.CompletedWorkers, append(testReq.FailedWorkers, "AssignmentFailed"))
		}
		return
	}

	if !resp.Accepted {
		log.Printf("Worker %s rejected test %s assignment: %s. Re-queueing test.", workerID, testReq.ID, resp.Message)
		uc.testRepo.AddFailedWorkerToTest(ctx, testReq.ID, workerID)
		select {
		case uc.testQueue <- testReq:
		default:
			log.Printf("Failed to re-queue test %s, test queue full. Marking test as failed.", testReq.ID)
			uc.testRepo.UpdateTestStatus(ctx, testReq.ID, "FAILED", testReq.CompletedWorkers, append(testReq.FailedWorkers, "WorkerRejected"))
		}
		return
	}

	log.Printf("Test %s assigned successfully to worker %s.", testReq.ID, workerID)

	// Add worker to assigned list only after successful assignment
	uc.testRepo.IncrementTestAssignedWorkers(ctx, testReq.ID, workerID)

	// Record the assignment for tracking
	uc.mu.Lock()
	if _, ok := uc.activeTestAssignments.Load(testReq.ID); !ok {
		uc.activeTestAssignments.Store(testReq.ID, make(map[string]bool))
	}
	if workersMap, ok := uc.activeTestAssignments.Load(testReq.ID); ok {
		workersMap.(map[string]bool)[workerID] = true
	}
	uc.mu.Unlock()
}

// TriggerAggregation manually triggers aggregation for a specific test.
func (uc *MasterUsecase) TriggerAggregation(ctx context.Context, testID string) {
	uc.aggregateTestResults(ctx, testID)
}

// aggregateTestResults fetches all raw results for a test and performs aggregation.
func (uc *MasterUsecase) aggregateTestResults(ctx context.Context, testID string) {
	log.Printf("Starting aggregation for test: %s", testID)
	results, err := uc.testResultRepo.GetResultsByTestID(ctx, testID)
	if err != nil {
		log.Printf("Error fetching raw results for aggregation for test %s: %v", testID, err)
		return
	}

	if len(results) == 0 {
		log.Printf("No raw results found for test %s to aggregate.", testID)
		return
	}

	// Simple aggregation logic (can be expanded)
	var totalRequests, successfulRequests, failedRequests, totalDuration int64
	var totalLatencyMs float64
	var p95Latencies []float64
	errorRates := make(map[string]int) // Map of error types/status codes to counts

	for _, res := range results {
		totalRequests += res.TotalRequests
		totalDuration += res.DurationMs
		successfulRequests += int64(res.SuccessRate * float64(res.TotalRequests)) // Approximate successful requests
		failedRequests += (res.TotalRequests - int64(res.SuccessRate*float64(res.TotalRequests)))

		totalLatencyMs += res.AverageLatencyMs * float64(res.CompletedRequests) // Weighted average
		p95Latencies = append(p95Latencies, res.P95LatencyMs)

		// Parse status codes
		for code, count := range res.StatusCodes {
			if code[0] != '2' { // Assuming 2xx are successful
				errorRates[code] += count
			}
		}
	}

	avgLatencyMs := 0.0
	if totalRequests > 0 {
		avgLatencyMs = totalLatencyMs / float64(totalRequests)
	}

	// Calculate overall P95 (simple median of P95s for now, more complex if using raw latencies)
	sort.Float64s(p95Latencies)
	p95LatencyMs := 0.0
	if len(p95Latencies) > 0 {
		p95LatencyMs = p95Latencies[int(0.95*float64(len(p95Latencies)))]
	}

	overallStatus := "COMPLETED_SUCCESS"
	if failedRequests > 0 {
		overallStatus = "COMPLETED_WITH_ERRORS"
	}

	// errorRatesJSON, _ := json.Marshal(errorRates)

	aggregatedResult := &domain.TestResultAggregated{
		TestID:             testID,
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		AvgLatencyMs:       avgLatencyMs,
		P95LatencyMs:       p95LatencyMs,
		ErrorRates:         errorRates,
		DurationMs:         totalDuration / int64(len(results)), // Average duration across workers
		OverallStatus:      overallStatus,
		CompletedAt:        time.Now(),
	}

	err = uc.aggregatedResultRepo.SaveAggregatedResult(ctx, aggregatedResult)
	if err != nil {
		log.Printf("Error saving aggregated result for test %s: %v", testID, err)
		return
	}
	log.Printf("Aggregated results saved for test: %s", testID)

	// Optionally, delete raw results to save space after aggregation
	// uc.testResultRepo.DeleteResultsByTestID(ctx, testID)
}

// GetDashboardStatus compiles and returns the current dashboard status.
func (uc *MasterUsecase) GetDashboardStatus(ctx context.Context) (*domain.DashboardStatus, error) {
	allWorkers, err := uc.workerRepo.GetAllWorkers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all workers for dashboard: %w", err)
	}

	totalWorkers := uint32(len(allWorkers))
	var availableWorkers uint32
	var busyWorkers uint32
	workerSummaries := make([]domain.WorkerSummary, 0, totalWorkers)

	for _, w := range allWorkers {
		if w.Status == "READY" {
			availableWorkers++
		} else if w.Status == "BUSY" {
			busyWorkers++
		}
		workerSummaries = append(workerSummaries, domain.WorkerSummary{
			WorkerID:          w.ID,
			StatusMessage:     w.LastProgressMessage,
			StatusType:        w.Status,
			CurrentTestID:     w.CurrentTestID,
			CompletedRequests: w.CompletedRequests,
			TotalRequests:     w.TotalRequests,
		})
	}

	allTests, err := uc.testRepo.GetAllTestRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tests for dashboard: %w", err)
	}

	activeTests := make([]domain.ActiveTestSummary, 0)
	for _, test := range allTests {
		if test.Status == "RUNNING" || test.Status == "PENDING" || test.Status == "PARTIALLY_FAILED" {
			// Calculate progress based on assigned workers vs completed/failed
			var progress float64
			if len(test.AssignedWorkersIDs) > 0 {
				progress = float64(len(test.CompletedWorkers)+len(test.FailedWorkers)) / float64(len(test.AssignedWorkersIDs))
			}

			// Aggregate request counts from worker summaries if available
			var totalReqsSent int64
			var totalReqsCompleted int64
			// var totalDurationMs int64 // Will be average duration from assigned workers

			// Iterate over active worker summaries to sum up progress for the current test
			for _, ws := range workerSummaries {
				if ws.CurrentTestID == test.ID {
					totalReqsSent += ws.TotalRequests
					totalReqsCompleted += ws.CompletedRequests
					// We don't have individual worker durations here, so this will be an approximation or
					// require fetching raw results to calculate more accurately.
					// For now, let's keep it simple or set to 0.
				}
			}

			activeTests = append(activeTests, domain.ActiveTestSummary{
				TestID:                 test.ID,
				TestName:               test.Name,
				AssignedWorkers:        uint32(len(test.AssignedWorkersIDs)),
				CompletedWorkers:       uint32(len(test.CompletedWorkers)),
				FailedWorkers:          uint32(len(test.FailedWorkers)),
				Status:                 test.Status,
				TotalRequestsSent:      totalReqsSent,
				TotalRequestsCompleted: totalReqsCompleted,
				TotalDurationMs:        0, // Placeholder, can be improved with more data
				Progress:               progress,
			})
		}
	}

	return &domain.DashboardStatus{
		TotalWorkers:     totalWorkers,
		AvailableWorkers: availableWorkers,
		BusyWorkers:      busyWorkers,
		ActiveTests:      activeTests,
		WorkerSummaries:  workerSummaries,
	}, nil
}

// GetAllTestRequests retrieves all stored test requests.
func (uc *MasterUsecase) GetAllTestRequests(ctx context.Context) ([]*domain.TestRequest, error) {
	return uc.testRepo.GetAllTestRequests(ctx)
}

// GetTestRequestsPaginated retrieves test requests with pagination.
func (uc *MasterUsecase) GetTestRequestsPaginated(ctx context.Context, limit, offset int) ([]*domain.TestRequest, int, error) {
	return uc.testRepo.GetTestRequestsPaginated(ctx, limit, offset)
}

// GetRawTestResults retrieves all raw test results for a given test ID.
func (uc *MasterUsecase) GetRawTestResults(ctx context.Context, testID string) ([]*domain.TestResult, error) {
	return uc.testResultRepo.GetResultsByTestID(ctx, testID)
}

// GetAggregatedTestResult retrieves the aggregated result for a given test ID.
func (uc *MasterUsecase) GetAggregatedTestResult(ctx context.Context, testID string) (*domain.TestResultAggregated, error) {
	return uc.aggregatedResultRepo.GetAggregatedResultByTestID(ctx, testID)
}

// cleanupStaleWorkers periodically checks for workers that haven't sent status updates
// and marks them as offline. It also re-queue tests if they were assigned to these workers.
func (uc *MasterUsecase) cleanupStaleWorkers(ctx context.Context) {
	log.Println("Running stale worker cleanup...")
	const staleThreshold = 30 * time.Second // Workers are considered stale if no update in 30 seconds

	allWorkers, err := uc.workerRepo.GetAllWorkers(ctx)
	if err != nil {
		log.Printf("Error fetching all workers for stale cleanup: %v", err)
		return
	}

	for _, worker := range allWorkers {
		if worker.Status != "OFFLINE" && time.Since(worker.LastSeen) > staleThreshold {
			log.Printf("Worker %s (%s) is stale. Marking offline.", worker.ID, worker.Address)
			err := uc.MarkWorkerOffline(ctx, worker.ID)
			if err != nil {
				log.Printf("Failed to mark stale worker %s offline: %v", worker.ID, err)
			}

			// If worker was busy, re-queue the test
			if worker.CurrentTestID != "" {
				test, err := uc.testRepo.GetTestRequestByID(ctx, worker.CurrentTestID)
				if err != nil {
					log.Printf("Could not retrieve test %s for stale worker %s cleanup: %v", worker.CurrentTestID, worker.ID, err)
					continue
				}
				// Only re-queue if the test is still running/pending and not fully completed/failed
				if test.Status == "RUNNING" || test.Status == "PENDING" {
					log.Printf("Re-queueing test %s as worker %s went offline.", test.ID, worker.ID)
					uc.testRepo.AddFailedWorkerToTest(ctx, test.ID, worker.ID) // Mark this worker as failed for this test
					select {
					case uc.testQueue <- test:
					default:
						log.Printf("Failed to re-queue test %s, test queue full.", test.ID)
					}
				}
			}
		}
	}
}

// StartAggregationBackgroundJob starts a background job that periodically checks for
// completed tests without aggregated results and processes them.
func (uc *MasterUsecase) StartAggregationBackgroundJob(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting aggregation background job with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Aggregation background job stopped due to context cancellation")
			return
		case <-ticker.C:
			uc.processOrphanedTests(ctx)
		}
	}
}

// processOrphanedTests finds completed tests without aggregated results and processes them.
func (uc *MasterUsecase) processOrphanedTests(ctx context.Context) {
	log.Println("Checking for completed tests without aggregated results...")

	// Query for completed tests that don't have aggregated results
	orphanedTests, err := uc.findCompletedTestsWithoutAggregation(ctx)
	if err != nil {
		log.Printf("Error finding orphaned tests: %v", err)
		return
	}

	if len(orphanedTests) == 0 {
		log.Println("No orphaned tests found")
		return
	}

	log.Printf("Found %d completed tests without aggregated results", len(orphanedTests))

	// Process each orphaned test
	for _, testID := range orphanedTests {
		log.Printf("Processing orphaned test: %s", testID)
		go uc.aggregateTestResults(ctx, testID)
	}
}

// findCompletedTestsWithoutAggregation queries the database to find completed tests
// that don't have corresponding aggregated results.
func (uc *MasterUsecase) findCompletedTestsWithoutAggregation(ctx context.Context) ([]string, error) {
	// Get all test requests to check their status
	allTests, err := uc.testRepo.GetAllTestRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all test requests: %w", err)
	}

	var orphanedTests []string
	for _, test := range allTests {
		// Only check completed tests
		if test.Status == "COMPLETED" || test.Status == "PARTIALLY_FAILED" || test.Status == "COMPLETED_WITH_ERRORS" {
			// Check if aggregated result exists
			_, err := uc.aggregatedResultRepo.GetAggregatedResultByTestID(ctx, test.ID)
			if err != nil {
				// If error contains "not found" or similar, this test needs aggregation
				// We'll assume any error means it doesn't exist for simplicity
				log.Printf("Aggregated result not found for completed test %s, adding to processing queue", test.ID)
				orphanedTests = append(orphanedTests, test.ID)
			}
		}
	}

	return orphanedTests, nil
}

// assignTestToMultipleWorkers distributes a test across multiple workers concurrently
func (uc *MasterUsecase) assignTestToMultipleWorkers(ctx context.Context, testReq *domain.TestRequest, workerIDs []string) {
	log.Printf("Assigning test %s to %d workers: %v (rate distribution: %s)",
		testReq.ID, len(workerIDs), workerIDs, testReq.RateDistribution)

	// Calculate how to distribute the load across workers based on distribution mode
	var workerRates []uint64
	var totalExpectedRate uint64

	switch testReq.RateDistribution {
	case "same":
		// Each worker gets the same full rate
		for i := 0; i < len(workerIDs); i++ {
			workerRates = append(workerRates, testReq.RatePerSecond)
		}
		totalExpectedRate = testReq.RatePerSecond * uint64(len(workerIDs))
		log.Printf("Using 'same' rate distribution: each worker gets %d req/s (total: %d req/s)",
			testReq.RatePerSecond, totalExpectedRate)

	case "weighted":
		// Distribute rate based on provided weights
		totalWeight := 0.0
		for _, weight := range testReq.RateWeights {
			totalWeight += weight
		}

		totalAssigned := uint64(0)
		for _, weight := range testReq.RateWeights {
			workerRate := uint64(float64(testReq.RatePerSecond) * weight / totalWeight)
			workerRates = append(workerRates, workerRate)
			totalAssigned += workerRate
		}

		// Handle rounding errors by adding remainder to the first worker
		if totalAssigned < testReq.RatePerSecond {
			remainder := testReq.RatePerSecond - totalAssigned
			workerRates[0] += remainder
		}

		totalExpectedRate = testReq.RatePerSecond
		log.Printf("Using 'weighted' rate distribution: weights %v, rates %v (total: %d req/s)",
			testReq.RateWeights, workerRates, totalExpectedRate)

	case "ramped":
		// Gradually increase rate across workers (first worker gets lower rate, last gets higher)
		baseRate := testReq.RatePerSecond / uint64(len(workerIDs))
		rampStep := baseRate / 2 // Ramp from 50% to 150% of base rate

		for i := 0; i < len(workerIDs); i++ {
			// Calculate ramped rate: starts at baseRate - rampStep, ends at baseRate + rampStep
			rampFactor := float64(i) / float64(len(workerIDs)-1) // 0.0 to 1.0
			workerRate := uint64(float64(baseRate) + (2.0*rampFactor-1.0)*float64(rampStep))
			if workerRate < 1 {
				workerRate = 1 // Minimum 1 req/s
			}
			workerRates = append(workerRates, workerRate)
		}

		totalExpectedRate = 0
		for _, rate := range workerRates {
			totalExpectedRate += rate
		}
		log.Printf("Using 'ramped' rate distribution: rates %v (total: %d req/s)",
			workerRates, totalExpectedRate)

	case "burst":
		// Concentrate higher load on first few workers, lower on the rest
		burstWorkers := len(workerIDs) / 2
		if burstWorkers < 1 {
			burstWorkers = 1
		}

		burstRate := (testReq.RatePerSecond * 70) / (100 * uint64(burstWorkers))                 // 70% of load on burst workers
		normalRate := (testReq.RatePerSecond * 30) / (100 * uint64(len(workerIDs)-burstWorkers)) // 30% on remaining

		for i := 0; i < len(workerIDs); i++ {
			if i < burstWorkers {
				workerRates = append(workerRates, burstRate)
			} else {
				workerRates = append(workerRates, normalRate)
			}
		}

		totalExpectedRate = 0
		for _, rate := range workerRates {
			totalExpectedRate += rate
		}
		log.Printf("Using 'burst' rate distribution: %d burst workers at %d req/s, %d normal workers at %d req/s (total: %d req/s)",
			burstWorkers, burstRate, len(workerIDs)-burstWorkers, normalRate, totalExpectedRate)

	default:
		// Default "shared" - divide the rate evenly across all workers
		baseRate := testReq.RatePerSecond / uint64(len(workerIDs))
		remainder := testReq.RatePerSecond % uint64(len(workerIDs))

		for i := 0; i < len(workerIDs); i++ {
			workerRate := baseRate
			if i < int(remainder) {
				workerRate++ // Distribute remainder among first workers
			}
			workerRates = append(workerRates, workerRate)
		}

		totalExpectedRate = testReq.RatePerSecond
		log.Printf("Using 'shared' rate distribution: rates %v (total: %d req/s)",
			workerRates, totalExpectedRate)
	}

	// Update test status to RUNNING - we'll add workers to assigned list after successful assignment
	uc.testRepo.UpdateTestStatus(ctx, testReq.ID, "RUNNING", nil, nil)

	// Initialize assignment tracking
	uc.mu.Lock()
	workersMap := make(map[string]bool)
	for _, workerID := range workerIDs {
		workersMap[workerID] = true
	}
	uc.activeTestAssignments.Store(testReq.ID, workersMap)
	uc.mu.Unlock()

	// Assign to each worker concurrently
	var wg sync.WaitGroup
	successfulAssignments := 0
	var assignmentMutex sync.Mutex

	for i, workerID := range workerIDs {
		wg.Add(1)
		go func(workerID string, workerIndex int) {
			defer wg.Done()

			// Get this worker's rate from the pre-calculated rates
			workerRate := workerRates[workerIndex]

			// Create a modified test request for this worker with its specific rate
			workerTestReq := *testReq
			workerTestReq.RatePerSecond = workerRate

			log.Printf("Assigning test %s to worker %s with rate %d req/s (mode: %s)",
				testReq.ID, workerID, workerRate, testReq.RateDistribution)

			connVal, ok := uc.activeWorkerClients.Load(workerID)
			if !ok {
				log.Printf("Worker %s connection not found during multi-worker assignment for test %s", workerID, testReq.ID)
				uc.MarkWorkerOffline(ctx, workerID)
				uc.testRepo.AddFailedWorkerToTest(ctx, testReq.ID, workerID)
				return
			}

			conn := connVal.(*grpc.ClientConn)
			client := pb.NewWorkerServiceClient(conn)

			// Mark worker as busy
			uc.workerRepo.UpdateWorkerStatus(ctx, workerID, "BUSY", testReq.ID,
				fmt.Sprintf("Running test (rate: %d req/s, mode: %s)", workerRate, testReq.RateDistribution), 0, 0)

			assignment := &pb.TestAssignment{
				TestId:            testReq.ID,
				VegetaPayloadJson: workerTestReq.VegetaPayloadJSON,
				DurationSeconds:   workerTestReq.DurationSeconds,
				RatePerSecond:     workerRate, // Use the distributed rate
				TargetsBase64:     workerTestReq.TargetsBase64,
			}

			assignmentCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			resp, err := client.AssignTest(assignmentCtx, assignment)
			if err != nil {
				log.Printf("Failed to assign test %s to worker %s: %v", testReq.ID, workerID, err)
				uc.MarkWorkerOffline(ctx, workerID)
				uc.testRepo.AddFailedWorkerToTest(ctx, testReq.ID, workerID)
				// Reset worker status back to READY if still reachable
				uc.workerRepo.UpdateWorkerStatus(ctx, workerID, "READY", "", "Assignment failed", 0, 0)
				return
			}

			if !resp.Accepted {
				log.Printf("Worker %s rejected test %s assignment: %s", workerID, testReq.ID, resp.Message)
				uc.testRepo.AddFailedWorkerToTest(ctx, testReq.ID, workerID)
				// Reset worker status back to READY since assignment failed
				uc.workerRepo.UpdateWorkerStatus(ctx, workerID, "READY", "", "Assignment rejected", 0, 0)
				// Add worker back to availability queue
				uc.addWorkerToAvailabilityQueue(workerID)
				return
			}

			log.Printf("Test %s assigned successfully to worker %s (rate: %d req/s, mode: %s)",
				testReq.ID, workerID, workerRate, testReq.RateDistribution)

			// Only add to assigned workers list after successful assignment
			uc.testRepo.IncrementTestAssignedWorkers(ctx, testReq.ID, workerID)

			assignmentMutex.Lock()
			successfulAssignments++
			assignmentMutex.Unlock()
		}(workerID, i)
	}

	// Wait for all assignments to complete
	wg.Wait()

	log.Printf("Multi-worker assignment completed for test %s: %d/%d workers assigned successfully",
		testReq.ID, successfulAssignments, len(workerIDs))

	// If no workers accepted the assignment, mark test as failed
	if successfulAssignments == 0 {
		log.Printf("No workers accepted test %s assignment, marking as failed", testReq.ID)
		uc.testRepo.UpdateTestStatus(ctx, testReq.ID, "FAILED",
			testReq.CompletedWorkers, append(testReq.FailedWorkers, "AllWorkersRejected"))
	}
}

// SaveWorkerTestResult saves a test result received from a worker to the database
func (uc *MasterUsecase) SaveWorkerTestResult(ctx context.Context, testResult *domain.TestResult) error {
	log.Printf("Saving test result from worker %s for test %s", testResult.WorkerID, testResult.TestID)

	// Save the test result to database
	err := uc.testResultRepo.SaveTestResult(ctx, testResult)
	if err != nil {
		log.Printf("Failed to save test result from worker %s for test %s: %v", testResult.WorkerID, testResult.TestID, err)
		return fmt.Errorf("failed to save test result: %w", err)
	}

	log.Printf("Successfully saved test result from worker %s for test %s (Total: %d, Completed: %d, Success Rate: %.2f%%)",
		testResult.WorkerID, testResult.TestID, testResult.TotalRequests, testResult.CompletedRequests, testResult.SuccessRate*100)

	// Mark this worker as completed in the test record
	err = uc.testRepo.AddCompletedWorkerToTest(ctx, testResult.TestID, testResult.WorkerID)
	if err != nil {
		log.Printf("Warning: Failed to mark worker %s as completed for test %s: %v", testResult.WorkerID, testResult.TestID, err)
	}

	// Check if all assigned workers have completed and update test status accordingly
	go func() {
		statusCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := uc.checkAndUpdateTestCompletion(statusCtx, testResult.TestID); err != nil {
			log.Printf("Warning: Failed to check test completion status for test %s: %v", testResult.TestID, err)
		}
	}()

	// Trigger aggregation asynchronously
	go func() {
		aggregateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := uc.updateAggregatedResult(aggregateCtx, testResult.TestID); err != nil {
			log.Printf("Warning: Failed to update aggregated result for test %s after receiving result from worker %s: %v",
				testResult.TestID, testResult.WorkerID, err)
		}
	}()

	return nil
}

// checkAndUpdateTestCompletion checks if all workers for a test have completed and updates the test status
func (uc *MasterUsecase) checkAndUpdateTestCompletion(ctx context.Context, testID string) error {
	// Get the test details
	test, err := uc.testRepo.GetTestRequestByID(ctx, testID)
	if err != nil {
		return fmt.Errorf("failed to get test %s: %w", testID, err)
	}

	// Skip if test is already marked as completed
	if test.Status == "COMPLETED" || test.Status == "FAILED" {
		return nil
	}

	totalAssigned := len(test.AssignedWorkersIDs)
	totalCompleted := len(test.CompletedWorkers)
	totalFailed := len(test.FailedWorkers)

	log.Printf("Test %s status check: Assigned=%d, Completed=%d, Failed=%d",
		testID, totalAssigned, totalCompleted, totalFailed)
	log.Printf("Test %s details: AssignedWorkers=%v, CompletedWorkers=%v, FailedWorkers=%v",
		testID, test.AssignedWorkersIDs, test.CompletedWorkers, test.FailedWorkers)

	// Check if all workers have finished (either completed or failed)
	if totalCompleted+totalFailed >= totalAssigned {
		var newStatus string
		if totalCompleted == totalAssigned {
			newStatus = "COMPLETED"
			log.Printf("✅ All workers completed successfully for test %s", testID)
		} else if totalCompleted > 0 {
			newStatus = "PARTIALLY_FAILED"
			log.Printf("⚠️ Test %s partially completed: %d succeeded, %d failed", testID, totalCompleted, totalFailed)
		} else {
			newStatus = "FAILED"
			log.Printf("❌ Test %s failed: all %d workers failed", testID, totalFailed)
		}

		// Update the test status
		err = uc.testRepo.UpdateTestStatus(ctx, testID, newStatus, test.CompletedWorkers, test.FailedWorkers)
		if err != nil {
			return fmt.Errorf("failed to update test %s status to %s: %w", testID, newStatus, err)
		}

		log.Printf("🎯 Updated test %s status to: %s", testID, newStatus)

		// Also update worker status back to READY
		for _, workerID := range test.AssignedWorkersIDs {
			err = uc.workerRepo.UpdateWorkerStatus(ctx, workerID, "READY", "", "Test completed", 0, 0)
			if err != nil {
				log.Printf("Warning: Failed to reset worker %s status to READY: %v", workerID, err)
			}
		}
	}

	return nil
}

// updateAggregatedResult recalculates and updates the aggregated result for a test
func (uc *MasterUsecase) updateAggregatedResult(ctx context.Context, testID string) error {
	// Get all results for this test
	results, err := uc.testResultRepo.GetResultsByTestID(ctx, testID)
	if err != nil {
		return fmt.Errorf("failed to get results for test %s: %w", testID, err)
	}

	if len(results) == 0 {
		return nil // No results to aggregate yet
	}
	// Calculate aggregated metrics
	var totalRequests, totalCompleted int64
	var totalDuration, totalLatency, totalP95 float64

	for _, result := range results {
		totalRequests += result.TotalRequests
		totalCompleted += result.CompletedRequests
		totalDuration += float64(result.DurationMs)
		totalLatency += result.AverageLatencyMs
		totalP95 += result.P95LatencyMs
	}

	numWorkers := len(results)
	aggregatedResult := &domain.TestResultAggregated{
		TestID:             testID,
		TotalRequests:      totalRequests,
		SuccessfulRequests: totalCompleted,
		FailedRequests:     totalRequests - totalCompleted,
		AvgLatencyMs:       totalLatency / float64(numWorkers),
		P95LatencyMs:       totalP95 / float64(numWorkers),
		DurationMs:         int64(totalDuration / float64(numWorkers)),
		OverallStatus:      "Completed",
		CompletedAt:        time.Now(),
	}

	// Save the aggregated result
	return uc.aggregatedResultRepo.SaveAggregatedResult(ctx, aggregatedResult)
}

// addWorkerToAvailabilityQueue safely adds a worker to the availability queue without duplicates
func (uc *MasterUsecase) addWorkerToAvailabilityQueue(workerID string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	// Check if worker is already in the queue
	if uc.availableWorkers[workerID] {
		log.Printf("Worker %s is already in availability queue, skipping duplicate addition", workerID)
		return
	}

	// Try to add to the channel (non-blocking)
	select {
	case uc.workerAvailability <- workerID:
		uc.availableWorkers[workerID] = true
		log.Printf("Worker %s added to availability queue (queue size: %d)", workerID, len(uc.availableWorkers))
	default:
		log.Printf("Worker availability queue full, %s not added immediately", workerID)
	}
}

// removeWorkerFromAvailabilityQueue removes a worker from tracking when assigned to a test
func (uc *MasterUsecase) removeWorkerFromAvailabilityQueue(workerID string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	delete(uc.availableWorkers, workerID)
	log.Printf("Worker %s removed from availability tracking (queue size: %d)", workerID, len(uc.availableWorkers))
}

// fixStuckTests detects and fixes tests that are stuck due to worker count mismatches
func (uc *MasterUsecase) fixStuckTests(ctx context.Context) {
	log.Println("Checking for stuck tests due to worker count mismatches...")

	// Get all running tests
	allTests, err := uc.testRepo.GetAllTestRequests(ctx)
	if err != nil {
		log.Printf("Error fetching tests for stuck test cleanup: %v", err)
		return
	}

	// Get total number of active workers
	allWorkers, err := uc.workerRepo.GetAllWorkers(ctx)
	if err != nil {
		log.Printf("Error fetching workers for stuck test cleanup: %v", err)
		return
	}

	activeWorkerCount := 0
	for _, worker := range allWorkers {
		if worker.Status != "OFFLINE" {
			activeWorkerCount++
		}
	}

	log.Printf("Active workers in system: %d", activeWorkerCount)

	for _, test := range allTests {
		if test.Status == "RUNNING" {
			totalAssigned := len(test.AssignedWorkersIDs)
			totalCompleted := len(test.CompletedWorkers)
			totalFailed := len(test.FailedWorkers)
			totalFinished := totalCompleted + totalFailed

			log.Printf("Checking stuck test %s: Assigned=%d, Completed=%d, Failed=%d, ActiveWorkers=%d",
				test.ID, totalAssigned, totalCompleted, totalFailed, activeWorkerCount)

			// Case 1: More workers assigned than exist in system
			if totalAssigned > activeWorkerCount {
				log.Printf("🔧 Test %s has %d assigned workers but only %d active workers exist - fixing assignment count",
					test.ID, totalAssigned, activeWorkerCount)

				// If all active workers have finished, complete the test
				if totalFinished >= activeWorkerCount {
					newStatus := "COMPLETED"
					if totalFailed > 0 {
						newStatus = "PARTIALLY_FAILED"
					}

					log.Printf("🔧 Completing stuck test %s (all %d active workers finished): %s",
						test.ID, activeWorkerCount, newStatus)

					err = uc.testRepo.UpdateTestStatus(ctx, test.ID, newStatus, test.CompletedWorkers, test.FailedWorkers)
					if err != nil {
						log.Printf("Error updating stuck test %s status: %v", test.ID, err)
					} else {
						log.Printf("✅ Fixed stuck test %s - status updated to %s", test.ID, newStatus)
					}
				}
			}

			// Case 2: Test has been running for too long (timeout)
			testAge := time.Since(test.CreatedAt)

			// Parse duration string (e.g., "10s", "5m")
			testDuration, err := time.ParseDuration(test.DurationSeconds)
			if err != nil {
				testDuration = 60 * time.Second // Default 60s if parse fails
			}
			maxTestDuration := testDuration + 5*time.Minute // Add 5 min buffer

			if testAge > maxTestDuration {
				log.Printf("🔧 Test %s has been running for %v (max: %v) - timing out",
					test.ID, testAge, maxTestDuration)

				newStatus := "PARTIALLY_FAILED"
				if totalCompleted == 0 {
					newStatus = "FAILED"
				}

				err = uc.testRepo.UpdateTestStatus(ctx, test.ID, newStatus, test.CompletedWorkers, test.FailedWorkers)
				if err != nil {
					log.Printf("Error timing out stuck test %s: %v", test.ID, err)
				} else {
					log.Printf("✅ Timed out stuck test %s - status updated to %s", test.ID, newStatus)
				}
			}
		}
	}
}

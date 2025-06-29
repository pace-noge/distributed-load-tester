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
	workerAvailability chan string // Channel for available worker IDs
	mu                 sync.Mutex  // Protects access to testQueue and workerAvailability
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
		workerAvailability:   make(chan string, 100),              // Buffered channel for available worker IDs
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
	select {
	case uc.workerAvailability <- worker.ID:
		log.Printf("Worker %s added to availability queue.", worker.ID)
	default:
		log.Printf("Worker availability queue full, %s not added immediately.", worker.ID)
	}
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
		select {
		case uc.workerAvailability <- workerID:
			log.Printf("Worker %s became READY, added to availability queue.", workerID)
		default:
			log.Printf("Worker availability queue full, %s not added immediately upon READY status.", workerID)
		}
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

	err := uc.testRepo.SaveTestRequest(ctx, testReq)
	if err != nil {
		return "", fmt.Errorf("failed to save test request: %w", err)
	}

	// Put test into queue for assignment
	select {
	case uc.testQueue <- testReq:
		log.Printf("Test %s submitted and added to assignment queue (requires %d workers).", testReq.ID, testReq.WorkerCount)
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

	// Update test status to RUNNING and assign worker
	uc.testRepo.IncrementTestAssignedWorkers(ctx, testReq.ID, workerID)
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

// HandleWorkerTestCompletion is called by the gRPC handler when a worker finishes or errors.
func (uc *MasterUsecase) HandleWorkerTestCompletion(ctx context.Context, testID, workerID string, isError bool) {
	log.Printf("Handling completion for test %s by worker %s. Is Error: %t", testID, workerID, isError)

	test, err := uc.testRepo.GetTestRequestByID(ctx, testID)
	if err != nil {
		log.Printf("Error getting test %s during completion handling: %v", testID, err)
		return
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	// Update test status (completed/failed worker)
	if isError {
		test.FailedWorkers = append(test.FailedWorkers, workerID)
		uc.testRepo.AddFailedWorkerToTest(ctx, testID, workerID)
	} else {
		test.CompletedWorkers = append(test.CompletedWorkers, workerID)
		uc.testRepo.AddCompletedWorkerToTest(ctx, testID, workerID)
	}

	// Remove assignment from tracking map
	if workersMapVal, ok := uc.activeTestAssignments.Load(testID); ok {
		if workersMap, ok := workersMapVal.(map[string]bool); ok {
			delete(workersMap, workerID)
		}
	}

	// Get updated test data to ensure we have the latest worker counts
	updatedTest, err := uc.testRepo.GetTestRequestByID(ctx, testID)
	if err != nil {
		log.Printf("Error getting updated test %s during completion handling: %v", testID, err)
		return
	}

	// Check if all workers assigned to this test have completed or failed
	totalExpectedWorkers := int(updatedTest.WorkerCount)
	totalCompletedWorkers := len(updatedTest.CompletedWorkers)
	totalFailedWorkers := len(updatedTest.FailedWorkers)
	totalFinishedWorkers := totalCompletedWorkers + totalFailedWorkers

	log.Printf("Test %s worker status: Expected=%d, Completed=%d, Failed=%d, Total Finished=%d",
		testID, totalExpectedWorkers, totalCompletedWorkers, totalFailedWorkers, totalFinishedWorkers)

	// Check if all expected workers have finished (either completed or failed)
	if totalFinishedWorkers >= totalExpectedWorkers {
		log.Printf("All %d workers for test %s have finished (completed: %d, failed: %d)",
			totalExpectedWorkers, testID, totalCompletedWorkers, totalFailedWorkers)

		// Determine final test status based on worker results
		var finalStatus string
		if totalFailedWorkers == 0 {
			finalStatus = "COMPLETED"
		} else if totalCompletedWorkers > 0 {
			finalStatus = "PARTIALLY_FAILED"
		} else {
			finalStatus = "FAILED"
		}

		log.Printf("Setting test %s final status to: %s", testID, finalStatus)
		uc.testRepo.UpdateTestStatus(ctx, testID, finalStatus, updatedTest.CompletedWorkers, updatedTest.FailedWorkers)

		// Trigger aggregation of results for this test
		log.Printf("Triggering aggregation for completed test %s", testID)
		go uc.aggregateTestResults(context.Background(), testID)
	} else {
		log.Printf("Test %s still waiting for %d more workers to finish",
			testID, totalExpectedWorkers-totalFinishedWorkers)
	}

	// Mark worker as READY again
	uc.workerRepo.UpdateWorkerStatus(ctx, workerID, "READY", "", "Ready for new tests", 0, 0)
	select {
	case uc.workerAvailability <- workerID:
		log.Printf("Worker %s became READY, added to availability queue.", workerID)
	default:
		log.Printf("Worker availability queue full, %s not added immediately upon READY status.", workerID)
	}
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
	log.Printf("Assigning test %s to %d workers: %v", testReq.ID, len(workerIDs), workerIDs)

	// Calculate how to distribute the load across workers
	// We'll divide the rate per second evenly across all workers
	baseRate := testReq.RatePerSecond / uint64(len(workerIDs))
	remainder := testReq.RatePerSecond % uint64(len(workerIDs))

	// Update test status to RUNNING and assign all workers
	testReq.AssignedWorkersIDs = workerIDs
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

			// Calculate this worker's rate (distribute remainder among first workers)
			workerRate := baseRate
			if workerIndex < int(remainder) {
				workerRate++
			}

			// Create a modified test request for this worker with its specific rate
			workerTestReq := *testReq
			workerTestReq.RatePerSecond = workerRate

			log.Printf("Assigning test %s to worker %s with rate %d req/s", testReq.ID, workerID, workerRate)

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
				fmt.Sprintf("Running test (rate: %d req/s)", workerRate), 0, 0)

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
				return
			}

			if !resp.Accepted {
				log.Printf("Worker %s rejected test %s assignment: %s", workerID, testReq.ID, resp.Message)
				uc.testRepo.AddFailedWorkerToTest(ctx, testReq.ID, workerID)
				return
			}

			log.Printf("Test %s assigned successfully to worker %s (rate: %d req/s)", testReq.ID, workerID, workerRate)

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

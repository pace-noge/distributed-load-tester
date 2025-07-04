package domain

import (
	"time"
)

// TestRequest represents a user-submitted load test configuration.
type TestRequest struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	VegetaPayloadJSON  string    `json:"vegetaPayloadJson"` // Raw JSON for Vegeta attack options
	DurationSeconds    string    `json:"durationSeconds"`   // e.g., "10s"
	RatePerSecond      uint64    `json:"ratePerSecond"`     // e.g., 50 for 50 req/s
	TargetsBase64      string    `json:"targetsBase64"`     // Base64 encoded targets content
	RequesterID        string    `json:"requesterId"`
	WorkerCount        uint32    `json:"workerCount"`        // Number of workers to use for this test
	RateDistribution   string    `json:"rateDistribution"`   // "shared", "same", "weighted", "ramped", or "burst" - how to distribute rate among workers
	RateWeights        []float64 `json:"rateWeights,omitempty"` // For "weighted" distribution: weight for each worker (optional)
	CreatedAt          time.Time `json:"createdAt"`
	Status             string    `json:"status"` // e.g., "PENDING", "RUNNING", "COMPLETED", "FAILED"
	AssignedWorkersIDs []string  `json:"assignedWorkersIds"`
	CompletedWorkers   []string  `json:"completedWorkers"`
	FailedWorkers      []string  `json:"failedWorkers"`
}

// TestResult represents the aggregated result of a single worker's test run.
type TestResult struct {
	ID                string         `json:"id"`
	TestID            string         `json:"testId"`
	WorkerID          string         `json:"workerId"`
	Metric            []byte         `json:"metric"` // Raw Vegeta Metric JSON or protobuf bytes
	Timestamp         time.Time      `json:"timestamp"`
	TotalRequests     int64          `json:"totalRequests"`
	CompletedRequests int64          `json:"completedRequests"`
	DurationMs        int64          `json:"durationMs"`
	SuccessRate       float64        `json:"successRate"`
	AverageLatencyMs  float64        `json:"averageLatencyMs"`
	P95LatencyMs      float64        `json:"p95LatencyMs"`
	StatusCodes       map[string]int `json:"statusCodes"` // Map of status code counts
}

// Worker represents a registered load testing worker.
type Worker struct {
	ID                  string    `json:"id"`
	Address             string    `json:"address"` // gRPC address (host:port)
	Status              string    `json:"status"`  // e.g., "READY", "BUSY", "OFFLINE"
	LastSeen            time.Time `json:"lastSeen"`
	CurrentTestID       string    `json:"currentTestId"`       // ID of the test it's currently running
	LastProgressMessage string    `json:"lastProgressMessage"` // Last progress message from worker
	CompletedRequests   int64     `json:"completedRequests"`
	TotalRequests       int64     `json:"totalRequests"`
}

// DashboardStatus provides a summary for the UI dashboard.
type DashboardStatus struct {
	TotalWorkers     uint32              `json:"total_workers"`
	AvailableWorkers uint32              `json:"available_workers"`
	BusyWorkers      uint32              `json:"busy_workers"`
	ActiveTests      []ActiveTestSummary `json:"active_tests"`
	WorkerSummaries  []WorkerSummary     `json:"worker_summaries"`
}

// ActiveTestSummary provides a summary of an ongoing or recently completed test.
type ActiveTestSummary struct {
	TestID                 string  `json:"test_id"`
	TestName               string  `json:"test_name"`
	AssignedWorkers        uint32  `json:"assigned_workers"`
	CompletedWorkers       uint32  `json:"completed_workers"`
	FailedWorkers          uint32  `json:"failed_workers"`
	Status                 string  `json:"status"`
	TotalRequestsSent      int64   `json:"total_requests_sent"`
	TotalRequestsCompleted int64   `json:"total_requests_completed"`
	TotalDurationMs        int64   `json:"total_duration_ms"`
	Progress               float64 `json:"progress"` // 0.0 - 1.0
}

// WorkerSummary provides a concise status of a worker for the dashboard.
type WorkerSummary struct {
	WorkerID          string `json:"worker_id"`
	StatusMessage     string `json:"status_message"`
	StatusType        string `json:"status_type"` // From proto enum, e.g., "READY", "BUSY"
	CurrentTestID     string `json:"current_test_id"`
	CompletedRequests int64  `json:"completed_requests"`
	TotalRequests     int64  `json:"total_requests"`
}

// TestResultAggregated represents a high-level aggregated view of a test result, for dashboard/reports
type TestResultAggregated struct {
	TestID             string         `json:"test_id"`
	TotalRequests      int64          `json:"total_requests"`
	SuccessfulRequests int64          `json:"successful_requests"`
	FailedRequests     int64          `json:"failed_requests"`
	AvgLatencyMs       float64        `json:"avg_latency_ms"`
	P95LatencyMs       float64        `json:"p95_latency_ms"`
	ErrorRates         map[string]int `json:"error_rates"` // Map of error types and counts
	DurationMs         int64          `json:"duration_ms"`
	OverallStatus      string         `json:"overall_status"` // "Success", "Partial Failure", "Failure"
	CompletedAt        time.Time      `json:"completed_at"`
}

type TestAssignment struct {
	TestID            string
	VegetaPayloadJSON string
	DurationSeconds   string
	RatePerSecond     uint64
	TargetsBase64     string
}

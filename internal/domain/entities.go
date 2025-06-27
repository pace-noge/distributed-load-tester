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
	CreatedAt          time.Time `json:"createdAt"`
	Status             string    `json:"status"` // e.g., "PENDING", "RUNNING", "COMPLETED", "FAILED"
	AssignedWorkersIDs []string  `json:"assignedWorkersIds"`
	CompletedWorkers   []string  `json:"completedWorkers"`
	FailedWorkers      []string  `json:"failedWorkers"`
}

// TestResult represents the aggregated result of a single worker's test run.
type TestResult struct {
	ID                string    `json:"id"`
	TestID            string    `json:"testId"`
	WorkerID          string    `json:"workerId"`
	Metric            []byte    `json:"metric"` // Raw Vegeta Metric JSON or protobuf bytes
	Timestamp         time.Time `json:"timestamp"`
	TotalRequests     int64     `json:"totalRequests"`
	CompletedRequests int64     `json:"completedRequests"`
	DurationMs        int64     `json:"durationMs"`
	SuccessRate       float64   `json:"successRate"`
	AverageLatencyMs  float64   `json:"averageLatencyMs"`
	P95LatencyMs      float64   `json:"p95LatencyMs"`
	StatusCodes       string    `json:"statusCodes"` // JSON string of status code counts
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
	TotalWorkers     uint32
	AvailableWorkers uint32
	BusyWorkers      uint32
	ActiveTests      []ActiveTestSummary
	WorkerSummaries  []WorkerSummary
}

// ActiveTestSummary provides a summary of an ongoing or recently completed test.
type ActiveTestSummary struct {
	TestID                 string
	TestName               string
	AssignedWorkers        uint32
	CompletedWorkers       uint32
	FailedWorkers          uint32
	Status                 string
	TotalRequestsSent      int64
	TotalRequestsCompleted int64
	TotalDurationMs        int64
	Progress               float64 // 0.0 - 1.0
}

// WorkerSummary provides a concise status of a worker for the dashboard.
type WorkerSummary struct {
	WorkerID          string
	StatusMessage     string
	StatusType        string // From proto enum, e.g., "READY", "BUSY"
	CurrentTestID     string
	CompletedRequests int64
	TotalRequests     int64
}

// TestResultAggregated represents a high-level aggregated view of a test result, for dashboard/reports
type TestResultAggregated struct {
	TestID             string    `json:"testId"`
	TotalRequests      int64     `json:"totalRequests"`
	SuccessfulRequests int64     `json:"successfulRequests"`
	FailedRequests     int64     `json:"failedRequests"`
	AvgLatencyMs       float64   `json:"avgLatencyMs"`
	P95LatencyMs       float64   `json:"p95LatencyMs"`
	ErrorRates         string    `json:"errorRates"` // JSON string of error types and counts
	DurationMs         int64     `json:"durationMs"`
	OverallStatus      string    `json:"overallStatus"` // "Success", "Partial Failure", "Failure"
	CompletedAt        time.Time `json:"completedAt"`
}

type TestAssignment struct {
	TestID            string
	VegetaPayloadJSON string
	DurationSeconds   string
	RatePerSecond     uint64
	TargetsBase64     string
}

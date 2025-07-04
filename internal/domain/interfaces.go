package domain

import (
	"context"
	"time"
)

// UserRepository defines operations for managing user information.
type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, userID string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, userID string, updates *UpdateUserRequest) (*User, error)
	UpdateUserPassword(ctx context.Context, userID string, hashedPassword string) error
	GetAllUsers(ctx context.Context) ([]*User, error)
	ActivateUser(ctx context.Context, userID string) error
	DeactivateUser(ctx context.Context, userID string) error
	UpdateLastLogin(ctx context.Context, userID string) error
}

// WorkerRepository defines operations for managing worker information.
type WorkerRepository interface {
	RegisterWorker(ctx context.Context, worker *Worker) error
	UpdateWorkerStatus(ctx context.Context, workerID string, status string, currentTestID string, progressMsg string, completedReqs, totalReqs int64) error
	GetWorkerByID(ctx context.Context, workerID string) (*Worker, error)
	GetAvailableWorkers(ctx context.Context) ([]*Worker, error)
	GetAllWorkers(ctx context.Context) ([]*Worker, error)
	MarkWorkerOffline(ctx context.Context, workerID string) error
}

// TestRepository defines operations for managing test requests and their states.
type TestRepository interface {
	SaveTestRequest(ctx context.Context, test *TestRequest) error
	UpdateTestStatus(ctx context.Context, testID string, status string, completedWorkers, failedWorkers []string) error
	GetTestRequestByID(ctx context.Context, testID string) (*TestRequest, error)
	GetAllTestRequests(ctx context.Context) ([]*TestRequest, error)
	GetTestRequestsPaginated(ctx context.Context, limit, offset int) ([]*TestRequest, int, error)
	GetTestsInRange(ctx context.Context, startDate, endDate time.Time) ([]*TestRequest, error)
	GetTestRequestsByUser(ctx context.Context, userID string) ([]*TestRequest, error)
	GetTestRequestsPaginatedByUser(ctx context.Context, userID string, limit, offset int) ([]*TestRequest, int, error)
	GetTestsInRangeByUser(ctx context.Context, userID string, startDate, endDate time.Time) ([]*TestRequest, error)
	// Add paginated per-user test history
	IncrementTestAssignedWorkers(ctx context.Context, testID string, workerID string) error
	AddCompletedWorkerToTest(ctx context.Context, testID string, workerID string) error
	AddFailedWorkerToTest(ctx context.Context, testID string, workerID string) error
}

// TestResultRepository defines operations for storing and retrieving raw test results.
type TestResultRepository interface {
	SaveTestResult(ctx context.Context, result *TestResult) error
	GetResultsByTestID(ctx context.Context, testID string) ([]*TestResult, error)
	DeleteResultsByTestID(ctx context.Context, testID string) error
}

// AggregatedResultRepository defines operations for storing and retrieving aggregated test results.
type AggregatedResultRepository interface {
	SaveAggregatedResult(ctx context.Context, result *TestResultAggregated) error
	GetAggregatedResultByTestID(ctx context.Context, testID string) (*TestResultAggregated, error)
	GetByTestID(ctx context.Context, testID string) (*TestResultAggregated, error) // Alias for GetAggregatedResultByTestID
	GetAllAggregatedResults(ctx context.Context) ([]*TestResultAggregated, error)
}

// VegetaExecutor defines operations for executing Vegeta load tests.
type VegetaExecutor interface {
	Attack(ctx context.Context, vegetaPayloadJSON, durationStr string, rate uint64, targetsBase64 string) (*TestResult, error)
}

// SharedLinkRepository defines operations for managing shared test links.
type SharedLinkRepository interface {
	CreateSharedLink(ctx context.Context, testID, sharedBy string, expiresAt time.Time) (*SharedLink, error)
	GetSharedLinkByID(ctx context.Context, linkID string) (*SharedLink, error)
	AddUsedBy(ctx context.Context, linkID, userID string) error
	GetInboxForUser(ctx context.Context, userID string) ([]*SharedLink, error)
	MarkInboxItemRead(ctx context.Context, linkID, userID string) error
}

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pace-noge/distributed-load-tester/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PostgresDB implements TestRepository, TestResultRepository, AggregatedResultRepository and WorkerRepository.
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB creates a new PostgreSQL database instance.
func NewPostgresDB(databaseURL string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL!")
	return &PostgresDB{db: db}, nil
}

// InitSchema creates the necessary tables if they don't exist.
func (p *PostgresDB) InitSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id VARCHAR(255) PRIMARY KEY,
            username VARCHAR(255) UNIQUE NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            first_name VARCHAR(255) NOT NULL,
            last_name VARCHAR(255) NOT NULL,
            role VARCHAR(50) NOT NULL DEFAULT 'user',
            is_active BOOLEAN NOT NULL DEFAULT true,
            created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            last_login_at TIMESTAMP WITH TIME ZONE
        );`,
		`CREATE TABLE IF NOT EXISTS workers (
            id VARCHAR(255) PRIMARY KEY,
            address VARCHAR(255) NOT NULL,
            status VARCHAR(50) NOT NULL,
            last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
            current_test_id VARCHAR(255) DEFAULT '',
            last_progress_message TEXT DEFAULT '',
            completed_requests BIGINT DEFAULT 0,
            total_requests BIGINT DEFAULT 0
        );`,
		`CREATE TABLE IF NOT EXISTS test_requests (
            id VARCHAR(255) PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            vegeta_payload_json TEXT NOT NULL,
            duration_seconds VARCHAR(50) NOT NULL,
            rate_per_second BIGINT NOT NULL,
            targets_base64 TEXT NOT NULL,
            requester_id VARCHAR(255) NOT NULL,
            worker_count INTEGER NOT NULL DEFAULT 1,
            created_at TIMESTAMP WITH TIME ZONE NOT NULL,
            status VARCHAR(50) NOT NULL,
            assigned_workers_ids TEXT[],
            completed_workers TEXT[],
            failed_workers TEXT[]
        );`,
		`CREATE TABLE IF NOT EXISTS test_results (
            id VARCHAR(255) PRIMARY KEY,
            test_id VARCHAR(255) NOT NULL,
            worker_id VARCHAR(255) NOT NULL,
            metric JSONB NOT NULL,
            timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
            total_requests BIGINT NOT NULL,
            completed_requests BIGINT NOT NULL,
            duration_ms BIGINT NOT NULL,
            success_rate DOUBLE PRECISION NOT NULL,
            average_latency_ms DOUBLE PRECISION NOT NULL,
            p95_latency_ms DOUBLE PRECISION NOT NULL,
            status_codes JSONB NOT NULL,
            FOREIGN KEY (test_id) REFERENCES test_requests(id) ON DELETE CASCADE
        );`,
		`CREATE TABLE IF NOT EXISTS aggregated_test_results (
            test_id VARCHAR(255) PRIMARY KEY,
            total_requests BIGINT NOT NULL,
            successful_requests BIGINT NOT NULL,
            failed_requests BIGINT NOT NULL,
            avg_latency_ms DOUBLE PRECISION NOT NULL,
            p95_latency_ms DOUBLE PRECISION NOT NULL,
            error_rates JSONB NOT NULL,
            duration_ms BIGINT NOT NULL,
            overall_status VARCHAR(50) NOT NULL,
            completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
            FOREIGN KEY (test_id) REFERENCES test_requests(id) ON DELETE CASCADE
        );`,
		`CREATE TABLE IF NOT EXISTS shared_links (
			id VARCHAR(255) PRIMARY KEY,
			test_id VARCHAR(255) NOT NULL,
			shared_by VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			used_by TEXT[],
			read_by TEXT[],
			FOREIGN KEY (test_id) REFERENCES test_requests(id) ON DELETE CASCADE
		);`,
		// Add worker_count column to existing test_requests table if it doesn't exist
		`ALTER TABLE test_requests ADD COLUMN IF NOT EXISTS worker_count INTEGER NOT NULL DEFAULT 1;`,
		// Create indexes for better performance
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);`,
	}

	for _, q := range queries {
		_, err := p.db.ExecContext(ctx, q)
		if err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}
	log.Println("PostgreSQL schema initialized successfully.")
	return nil
}

// Close closes the database connection.
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// GetDB returns the underlying sql.DB instance
func (p *PostgresDB) GetDB() *sql.DB {
	return p.db
}

// --- WorkerRepository Implementations ---

// RegisterWorker registers or updates a worker's initial status.
func (p *PostgresDB) RegisterWorker(ctx context.Context, worker *domain.Worker) error {
	query := `INSERT INTO workers (id, address, status, last_seen)
              VALUES ($1, $2, $3, $4)
              ON CONFLICT (id) DO UPDATE
              SET address = EXCLUDED.address, status = EXCLUDED.status, last_seen = EXCLUDED.last_seen;`
	_, err := p.db.ExecContext(ctx, query, worker.ID, worker.Address, worker.Status, worker.LastSeen)
	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}
	return nil
}

// UpdateWorkerStatus updates a worker's status and progress.
func (p *PostgresDB) UpdateWorkerStatus(ctx context.Context, workerID string, status string, currentTestID string, progressMsg string, completedReqs, totalReqs int64) error {
	query := `UPDATE workers SET status = $1, last_seen = $2, current_test_id = $3, last_progress_message = $4, completed_requests = $5, total_requests = $6 WHERE id = $7;`
	_, err := p.db.ExecContext(ctx, query, status, time.Now(), currentTestID, progressMsg, completedReqs, totalReqs, workerID)
	if err != nil {
		return fmt.Errorf("failed to update worker status: $w", err)
	}
	return nil
}

// GetWorkerByID retrieves a worker by its ID.
func (p *PostgresDB) GetWorkerByID(ctx context.Context, workerID string) (*domain.Worker, error) {
	worker := &domain.Worker{}
	query := `SELECT id, address, status, last_seen, current_test_id, last_progress_message, completed_requests, total_requests FROM workers WHERE id = $1;`
	err := p.db.QueryRowContext(ctx, query, workerID).Scan(
		&worker.ID, &worker.Address, &worker.Status, &worker.LastSeen, &worker.CurrentTestID,
		&worker.LastProgressMessage, &worker.CompletedRequests, &worker.TotalRequests,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("worker not found: %s", workerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get worker by ID: %w", err)
	}
	return worker, nil
}

// GetAvailableWorkers retrieves all workers with 'READY' status.
func (p *PostgresDB) GetAvailableWorkers(ctx context.Context) ([]*domain.Worker, error) {
	query := `SELECT id, address, status, last_seen, current_test_id, last_progress_message, completed_requests, total_requests FROM workers WHERE status = 'READY';`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get available workers: %w", err)
	}
	defer rows.Close()

	var workers []*domain.Worker
	for rows.Next() {
		worker := &domain.Worker{}
		err := rows.Scan(
			&worker.ID, &worker.Address, &worker.Status, &worker.LastSeen, &worker.CurrentTestID,
			&worker.LastProgressMessage, &worker.CompletedRequests, &worker.TotalRequests,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker row: %w", err)
		}
		workers = append(workers, worker)
	}
	return workers, nil
}

// GetAllWorkers retrieves all registered workers.
func (p *PostgresDB) GetAllWorkers(ctx context.Context) ([]*domain.Worker, error) {
	query := `SELECT id, address, status, last_seen, current_test_id, last_progress_message, completed_requests, total_requests FROM workers;`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all workers: %w", err)
	}
	defer rows.Close()

	var workers []*domain.Worker
	for rows.Next() {
		worker := &domain.Worker{}
		err := rows.Scan(
			&worker.ID, &worker.Address, &worker.Status, &worker.LastSeen, &worker.CurrentTestID,
			&worker.LastProgressMessage, &worker.CompletedRequests, &worker.TotalRequests,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker row: %w", err)
		}
		workers = append(workers, worker)
	}
	return workers, nil
}

// MarkWorkerOffline updates a worker's status to OFFLINE.
func (p *PostgresDB) MarkWorkerOffline(ctx context.Context, workerID string) error {
	query := `UPDATE workers SET status = 'OFFLINE', last_seen = $1 WHERE id = $2;`
	_, err := p.db.ExecContext(ctx, query, time.Now(), workerID)
	if err != nil {
		return fmt.Errorf("failed to mark worker offline: %w", err)
	}
	return nil
}

// --- TestRepository Implementations ---

// SaveTestRequest saves a new test request.
func (p *PostgresDB) SaveTestRequest(ctx context.Context, test *domain.TestRequest) error {
	if test.ID == "" {
		test.ID = uuid.New().String()
	}
	if test.CreatedAt.IsZero() {
		test.CreatedAt = time.Now()
	}
	if test.Status == "" {
		test.Status = "PENDING"
	}

	query := `INSERT INTO test_requests (id, name, vegeta_payload_json, duration_seconds, rate_per_second, targets_base64, requester_id, worker_count, created_at, status, assigned_workers_ids, completed_workers, failed_workers)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);`
	_, err := p.db.ExecContext(ctx, query, test.ID, test.Name, test.VegetaPayloadJSON, test.DurationSeconds,
		test.RatePerSecond, test.TargetsBase64, test.RequesterID, test.WorkerCount, test.CreatedAt, test.Status,
		pq.Array(test.AssignedWorkersIDs), pq.Array(test.CompletedWorkers), pq.Array(test.FailedWorkers))
	if err != nil {
		return fmt.Errorf("failed to save test request: %w", err)
	}
	return nil
}

// UpdateTestStatus updates the status of a test request.
func (p *PostgresDB) UpdateTestStatus(ctx context.Context, testID string, status string, completedWorkers, failedWorkers []string) error {
	query := `UPDATE test_requests SET status = $1, completed_workers = $2, failed_workers = $3 WHERE id = $4;`
	_, err := p.db.ExecContext(ctx, query, status, pq.Array(completedWorkers), pq.Array(failedWorkers), testID)
	if err != nil {
		return fmt.Errorf("failed to update test status: %w", err)
	}
	return nil
}

// GetTestRequestByID retrieves a test request by its ID.
func (p *PostgresDB) GetTestRequestByID(ctx context.Context, testID string) (*domain.TestRequest, error) {
	test := &domain.TestRequest{}
	query := `SELECT id, name, vegeta_payload_json, duration_seconds, rate_per_second, targets_base64, requester_id, worker_count, created_at, status, assigned_workers_ids, completed_workers, failed_workers FROM test_requests WHERE id = $1;`
	err := p.db.QueryRowContext(ctx, query, testID).Scan(
		&test.ID, &test.Name, &test.VegetaPayloadJSON, &test.DurationSeconds, &test.RatePerSecond, &test.TargetsBase64,
		&test.RequesterID, &test.WorkerCount, &test.CreatedAt, &test.Status, pq.Array(&test.AssignedWorkersIDs), pq.Array(&test.CompletedWorkers), pq.Array(&test.FailedWorkers),
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("test request not found: %s", testID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get test request by ID: %w", err)
	}
	return test, nil
}

// GetAllTestRequests retrieves all test requests.
func (p *PostgresDB) GetAllTestRequests(ctx context.Context) ([]*domain.TestRequest, error) {
	query := `SELECT id, name, vegeta_payload_json, duration_seconds, rate_per_second, targets_base64, requester_id, worker_count, created_at, status, assigned_workers_ids, completed_workers, failed_workers FROM test_requests ORDER BY created_at DESC;`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all test requests: %w", err)
	}
	defer rows.Close()

	var tests []*domain.TestRequest
	for rows.Next() {
		test := &domain.TestRequest{}
		err := rows.Scan(
			&test.ID, &test.Name, &test.VegetaPayloadJSON, &test.DurationSeconds, &test.RatePerSecond, &test.TargetsBase64,
			&test.RequesterID, &test.WorkerCount, &test.CreatedAt, &test.Status, pq.Array(&test.AssignedWorkersIDs), pq.Array(&test.CompletedWorkers), pq.Array(&test.FailedWorkers),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test request row: %w", err)
		}
		tests = append(tests, test)
	}
	return tests, nil
}

// GetTestRequestsPaginated retrieves test requests with pagination.
func (p *PostgresDB) GetTestRequestsPaginated(ctx context.Context, limit, offset int) ([]*domain.TestRequest, int, error) {
	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM test_requests`
	err := p.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get paginated results
	query := `SELECT id, name, vegeta_payload_json, duration_seconds, rate_per_second, targets_base64, requester_id, worker_count, created_at, status, assigned_workers_ids, completed_workers, failed_workers
		FROM test_requests
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := p.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get paginated test requests: %w", err)
	}
	defer rows.Close()

	var tests []*domain.TestRequest
	for rows.Next() {
		test := &domain.TestRequest{}
		err := rows.Scan(
			&test.ID, &test.Name, &test.VegetaPayloadJSON, &test.DurationSeconds, &test.RatePerSecond, &test.TargetsBase64,
			&test.RequesterID, &test.WorkerCount, &test.CreatedAt, &test.Status, pq.Array(&test.AssignedWorkersIDs), pq.Array(&test.CompletedWorkers), pq.Array(&test.FailedWorkers),
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan test request row: %w", err)
		}
		tests = append(tests, test)
	}

	return tests, totalCount, nil
}

// GetTestRequestsPaginatedByUser retrieves test requests for a specific user with pagination.
func (p *PostgresDB) GetTestRequestsPaginatedByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.TestRequest, int, error) {
	// Get total count for this user
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM test_requests WHERE requester_id = $1`
	err := p.db.QueryRowContext(ctx, countQuery, userID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count for user: %w", err)
	}

	// Get paginated results for this user
	query := `SELECT id, name, vegeta_payload_json, duration_seconds, rate_per_second, targets_base64, requester_id, worker_count, created_at, status, assigned_workers_ids, completed_workers, failed_workers
		FROM test_requests
		WHERE requester_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := p.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get paginated test requests by user: %w", err)
	}
	defer rows.Close()

	var tests []*domain.TestRequest
	for rows.Next() {
		test := &domain.TestRequest{}
		err := rows.Scan(
			&test.ID, &test.Name, &test.VegetaPayloadJSON, &test.DurationSeconds, &test.RatePerSecond, &test.TargetsBase64,
			&test.RequesterID, &test.WorkerCount, &test.CreatedAt, &test.Status, pq.Array(&test.AssignedWorkersIDs), pq.Array(&test.CompletedWorkers), pq.Array(&test.FailedWorkers),
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan test request row: %w", err)
		}
		tests = append(tests, test)
	}

	return tests, totalCount, nil
}

// IncrementTestAssignedWorkers appends a worker ID to the assigned_workers_ids array.
func (p *PostgresDB) IncrementTestAssignedWorkers(ctx context.Context, testID string, workerID string) error {
	query := `UPDATE test_requests SET assigned_workers_ids = array_append(assigned_workers_ids, $1) WHERE id = $2;`
	_, err := p.db.ExecContext(ctx, query, workerID, testID)
	if err != nil {
		return fmt.Errorf("failed to increment assigned workers for test %s: %w", testID, err)
	}
	return nil
}

// AddCompletedWorkerToTest adds a worker ID to the completed_workers array.
func (p *PostgresDB) AddCompletedWorkerToTest(ctx context.Context, testID string, workerID string) error {
	query := `UPDATE test_requests SET completed_workers = array_append(completed_workers, $1) WHERE id = $2;`
	_, err := p.db.ExecContext(ctx, query, workerID, testID)
	if err != nil {
		return fmt.Errorf("failed to add completed worker to test %s: %w", testID, err)
	}
	return nil
}

// AddFailedWorkerToTest adds a worker ID to the failed_workers array.
func (p *PostgresDB) AddFailedWorkerToTest(ctx context.Context, testID string, workerID string) error {
	query := `UPDATE test_requests SET failed_workers = array_append(failed_workers, $1) WHERE id = $2;`
	_, err := p.db.ExecContext(ctx, query, workerID, testID)
	if err != nil {
		return fmt.Errorf("failed to add failed worker to test %s: %w", testID, err)
	}
	return nil
}

// --- TestResultRepository Implementations ---

// SaveTestResult saves a single worker's test result.
func (p *PostgresDB) SaveTestResult(ctx context.Context, result *domain.TestResult) error {
	if result.ID == "" {
		result.ID = uuid.New().String()
	}
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}

	statusCodeJSON, err := json.Marshal(result.StatusCodes)
	if err != nil {
		return fmt.Errorf("failed to marshal status codes: %w", err)
	}

	query := `INSERT INTO test_results (id, test_id, worker_id, metric, timestamp, total_requests, completed_requests, duration_ms, success_rate, average_latency_ms, p95_latency_ms, status_codes)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);`
	_, err = p.db.ExecContext(ctx, query, result.ID, result.TestID, result.WorkerID, result.Metric, result.Timestamp,
		result.TotalRequests, result.CompletedRequests, result.DurationMs, result.SuccessRate, result.AverageLatencyMs,
		result.P95LatencyMs, statusCodeJSON)
	if err != nil {
		return fmt.Errorf("failed to save test result: %w", err)
	}
	return nil
}

// GetResultsByTestID retrieves all raw test results for a given test ID.
func (p *PostgresDB) GetResultsByTestID(ctx context.Context, testID string) ([]*domain.TestResult, error) {
	query := `SELECT id, test_id, worker_id, metric, timestamp, total_requests, completed_requests, duration_ms, success_rate, average_latency_ms, p95_latency_ms, status_codes FROM test_results WHERE test_id = $1 ORDER BY timestamp ASC;`
	rows, err := p.db.QueryContext(ctx, query, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results by test ID: %w", err)
	}
	defer rows.Close()

	var results []*domain.TestResult
	for rows.Next() {
		result := &domain.TestResult{}
		var metricJSON, statusCodeJSON []byte
		err := rows.Scan(
			&result.ID, &result.TestID, &result.WorkerID, &metricJSON, &result.Timestamp,
			&result.TotalRequests, &result.CompletedRequests, &result.DurationMs, &result.SuccessRate,
			&result.AverageLatencyMs, &result.P95LatencyMs, &statusCodeJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test result row: %w", err)
		}
		result.Metric = metricJSON

		err = json.Unmarshal(statusCodeJSON, &result.StatusCodes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal status codes: %w", err)
		}

		results = append(results, result)
	}
	return results, nil
}

// DeleteResultsByTestID deletes all raw test results for a given test ID.
func (p *PostgresDB) DeleteResultsByTestID(ctx context.Context, testID string) error {
	query := `DELETE FROM test_results WHERE test_id = $1;`
	_, err := p.db.ExecContext(ctx, query, testID)
	if err != nil {
		return fmt.Errorf("failed to delete test results by ID: %w", err)
	}
	return nil
}

// --- AggregatedResultRepository Implementations ---

// SaveAggregatedResult saves an aggregated test result.
func (p *PostgresDB) SaveAggregatedResult(ctx context.Context, result *domain.TestResultAggregated) error {
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now()
	}

	errorRatesJSON, err := json.Marshal(result.ErrorRates)
	if err != nil {
		return fmt.Errorf("failed to marshal error rates: %w", err)
	}

	query := `INSERT INTO aggregated_test_results (test_id, total_requests, successful_requests, failed_requests, avg_latency_ms, p95_latency_ms, error_rates, duration_ms, overall_status, completed_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
              ON CONFLICT (test_id) DO UPDATE SET
              total_requests = EXCLUDED.total_requests,
              successful_requests = EXCLUDED.successful_requests,
              failed_requests = EXCLUDED.failed_requests,
              avg_latency_ms = EXCLUDED.avg_latency_ms,
              p95_latency_ms = EXCLUDED.p95_latency_ms,
              error_rates = EXCLUDED.error_rates,
              duration_ms = EXCLUDED.duration_ms,
              overall_status = EXCLUDED.overall_status,
              completed_at = EXCLUDED.completed_at;` // Update on conflict to handle re-aggregation
	_, err = p.db.ExecContext(ctx, query, result.TestID, result.TotalRequests, result.SuccessfulRequests,
		result.FailedRequests, result.AvgLatencyMs, result.P95LatencyMs, errorRatesJSON,
		result.DurationMs, result.OverallStatus, result.CompletedAt)
	if err != nil {
		return fmt.Errorf("failed to save aggregated test result: %w", err)
	}
	return nil
}

// GetAggregatedResultByTestID retrieves an aggregated test result by its ID.
func (p *PostgresDB) GetAggregatedResultByTestID(ctx context.Context, testID string) (*domain.TestResultAggregated, error) {
	if testID == "" {
		return nil, fmt.Errorf("test ID cannot be empty")
	}

	result := &domain.TestResultAggregated{}
	var errorRatesJSON []byte
	query := `SELECT test_id, total_requests, successful_requests, failed_requests, avg_latency_ms, p95_latency_ms, error_rates, duration_ms, overall_status, completed_at FROM aggregated_test_results WHERE test_id = $1;`
	err := p.db.QueryRowContext(ctx, query, testID).Scan(
		&result.TestID, &result.TotalRequests, &result.SuccessfulRequests, &result.FailedRequests,
		&result.AvgLatencyMs, &result.P95LatencyMs, &errorRatesJSON, &result.DurationMs,
		&result.OverallStatus, &result.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("aggregated test result not found for test ID: %s", testID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated test result by ID: %w", err)
	}

	err = json.Unmarshal(errorRatesJSON, &result.ErrorRates)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal error rates: %w", err)
	}

	return result, nil
}

// GetAllAggregatedResults retrieves all aggregated test results.
func (p *PostgresDB) GetAllAggregatedResults(ctx context.Context) ([]*domain.TestResultAggregated, error) {
	query := `SELECT test_id, total_requests, successful_requests, failed_requests, avg_latency_ms, p95_latency_ms, error_rates, duration_ms, overall_status, completed_at FROM aggregated_test_results ORDER BY completed_at DESC;`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all aggregated test results: %w", err)
	}
	defer rows.Close()

	var results []*domain.TestResultAggregated
	for rows.Next() {
		result := &domain.TestResultAggregated{}
		var errorRatesJSON []byte
		err := rows.Scan(
			&result.TestID, &result.TotalRequests, &result.SuccessfulRequests, &result.FailedRequests,
			&result.AvgLatencyMs, &result.P95LatencyMs, &errorRatesJSON, &result.DurationMs,
			&result.OverallStatus, &result.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregated test result row: %w", err)
		}

		err = json.Unmarshal(errorRatesJSON, &result.ErrorRates)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal error rates: %w", err)
		}

		results = append(results, result)
	}
	return results, nil
}

// GetTestsInRange retrieves test requests within a date range
func (p *PostgresDB) GetTestsInRange(ctx context.Context, startDate, endDate time.Time) ([]*domain.TestRequest, error) {
	query := `SELECT id, name, vegeta_payload_json, duration_seconds, rate_per_second, targets_base64, requester_id, worker_count, created_at, status, assigned_workers_ids, completed_workers, failed_workers
              FROM test_requests
              WHERE created_at >= $1 AND created_at <= $2
              ORDER BY created_at DESC;`

	rows, err := p.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get test requests in range: %w", err)
	}
	defer rows.Close()

	var tests []*domain.TestRequest
	for rows.Next() {
		test := &domain.TestRequest{}
		err := rows.Scan(
			&test.ID, &test.Name, &test.VegetaPayloadJSON, &test.DurationSeconds, &test.RatePerSecond, &test.TargetsBase64,
			&test.RequesterID, &test.WorkerCount, &test.CreatedAt, &test.Status, pq.Array(&test.AssignedWorkersIDs), pq.Array(&test.CompletedWorkers), pq.Array(&test.FailedWorkers),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test request row: %w", err)
		}
		tests = append(tests, test)
	}
	return tests, nil
}

// GetTestRequestsByUser retrieves all test requests for a specific user.
func (p *PostgresDB) GetTestRequestsByUser(ctx context.Context, userID string) ([]*domain.TestRequest, error) {
	query := `SELECT id, name, vegeta_payload_json, duration_seconds, rate_per_second, targets_base64, requester_id, worker_count, created_at, status, assigned_workers_ids, completed_workers, failed_workers FROM test_requests WHERE requester_id = $1 ORDER BY created_at DESC;`
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test requests by user: %w", err)
	}
	defer rows.Close()

	var tests []*domain.TestRequest
	for rows.Next() {
		test := &domain.TestRequest{}
		err := rows.Scan(
			&test.ID, &test.Name, &test.VegetaPayloadJSON, &test.DurationSeconds, &test.RatePerSecond, &test.TargetsBase64,
			&test.RequesterID, &test.WorkerCount, &test.CreatedAt, &test.Status, pq.Array(&test.AssignedWorkersIDs), pq.Array(&test.CompletedWorkers), pq.Array(&test.FailedWorkers),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test request row: %w", err)
		}
		tests = append(tests, test)
	}
	return tests, nil
}

// GetTestsInRangeByUser retrieves test requests for a user in a date range.
func (p *PostgresDB) GetTestsInRangeByUser(ctx context.Context, userID string, startDate, endDate time.Time) ([]*domain.TestRequest, error) {
	query := `SELECT id, name, vegeta_payload_json, duration_seconds, rate_per_second, targets_base64, requester_id, worker_count, created_at, status, assigned_workers_ids, completed_workers, failed_workers FROM test_requests WHERE requester_id = $1 AND created_at >= $2 AND created_at <= $3 ORDER BY created_at DESC;`
	rows, err := p.db.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get test requests by user in range: %w", err)
	}
	defer rows.Close()

	var tests []*domain.TestRequest
	for rows.Next() {
		test := &domain.TestRequest{}
		err := rows.Scan(
			&test.ID, &test.Name, &test.VegetaPayloadJSON, &test.DurationSeconds, &test.RatePerSecond, &test.TargetsBase64,
			&test.RequesterID, &test.WorkerCount, &test.CreatedAt, &test.Status, pq.Array(&test.AssignedWorkersIDs), pq.Array(&test.CompletedWorkers), pq.Array(&test.FailedWorkers),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test request row: %w", err)
		}
		tests = append(tests, test)
	}
	return tests, nil
}

// GetByTestID is an alias for GetAggregatedResultByTestID for consistency
func (p *PostgresDB) GetByTestID(ctx context.Context, testID string) (*domain.TestResultAggregated, error) {
	return p.GetAggregatedResultByTestID(ctx, testID)
}

// SharedLinkRepository implementation
func (p *PostgresDB) CreateSharedLink(ctx context.Context, testID, sharedBy string, expiresAt time.Time) (*domain.SharedLink, error) {
	id := uuid.New().String()
	_, err := p.db.ExecContext(ctx, `INSERT INTO shared_links (id, test_id, shared_by, created_at, expires_at, used_by, read_by) VALUES ($1, $2, $3, NOW(), $4, $5, $6)`,
		id, testID, sharedBy, expiresAt, pq.Array([]string{}), pq.Array([]string{}))
	if err != nil {
		return nil, fmt.Errorf("failed to create shared link: %w", err)
	}
	return &domain.SharedLink{
		ID:        id,
		TestID:    testID,
		SharedBy:  sharedBy,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		UsedBy:    []string{},
	}, nil
}

func (p *PostgresDB) GetSharedLinkByID(ctx context.Context, linkID string) (*domain.SharedLink, error) {
	row := p.db.QueryRowContext(ctx, `SELECT id, test_id, shared_by, created_at, expires_at, used_by FROM shared_links WHERE id = $1`, linkID)
	var link domain.SharedLink
	var usedBy []string
	if err := row.Scan(&link.ID, &link.TestID, &link.SharedBy, &link.CreatedAt, &link.ExpiresAt, pq.Array(&usedBy)); err != nil {
		return nil, fmt.Errorf("failed to get shared link: %w", err)
	}
	link.UsedBy = usedBy
	link.IsExpired = time.Now().After(link.ExpiresAt)
	return &link, nil
}

func (p *PostgresDB) AddUsedBy(ctx context.Context, linkID, userID string) error {
	_, err := p.db.ExecContext(ctx, `UPDATE shared_links SET used_by = array_append(used_by, $1) WHERE id = $2 AND NOT (used_by @> ARRAY[$1])`, userID, linkID)
	return err
}

func (p *PostgresDB) GetInboxForUser(ctx context.Context, userID string) ([]*domain.SharedLink, error) {
	rows, err := p.db.QueryContext(ctx, `SELECT id, test_id, shared_by, created_at, expires_at, used_by, read_by FROM shared_links WHERE used_by @> ARRAY[$1]`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var inbox []*domain.SharedLink
	for rows.Next() {
		var link domain.SharedLink
		var usedBy, readBy []string
		err := rows.Scan(&link.ID, &link.TestID, &link.SharedBy, &link.CreatedAt, &link.ExpiresAt, pq.Array(&usedBy), pq.Array(&readBy))
		if err != nil {
			return nil, err
		}
		link.UsedBy = usedBy
		link.IsExpired = time.Now().After(link.ExpiresAt)
		inbox = append(inbox, &link)
	}
	return inbox, nil
}

func (p *PostgresDB) MarkInboxItemRead(ctx context.Context, linkID, userID string) error {
	_, err := p.db.ExecContext(ctx, `UPDATE shared_links SET read_by = array_append(read_by, $1) WHERE id = $2 AND NOT (read_by @> ARRAY[$1])`, userID, linkID)
	return err
}

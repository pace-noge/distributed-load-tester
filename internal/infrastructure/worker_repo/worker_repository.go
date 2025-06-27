package worker_repo

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
)

// InMemoryWorkerRepository implements the domain.WorkerRepository interface
// using an in-memory map. This is suitable for managing active worker states
// in a scenario where master might restart (and workers would re-register).
// For truly persistent worker registration across Master restarts, a database
// implementation would be required, but real-time status updates are best kept in memory.
type InMemoryWorkerRepository struct {
	workers map[string]*domain.Worker
	mu      sync.RWMutex
}

// NewInMemoryWorkerRepository creates a new InMemoryWorkerRepository.
func NewInMemoryWorkerRepository() *InMemoryWorkerRepository {
	return &InMemoryWorkerRepository{
		workers: make(map[string]*domain.Worker),
	}
}

// RegisterWorker adds or updates a worker in memory.
func (r *InMemoryWorkerRepository) RegisterWorker(ctx context.Context, worker *domain.Worker) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	worker.LastSeen = time.Now()
	r.workers[worker.ID] = worker
	log.Printf("Worker %s registered/updated in-memory.", worker.ID)
	return nil
}

// UpdateWorkerStatus updates a worker's status and progress in memory.
func (r *InMemoryWorkerRepository) UpdateWorkerStatus(ctx context.Context, workerID string, status string, currentTestID string, progressMsg string, completedReqs, totalReqs int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if worker, ok := r.workers[workerID]; ok {
		worker.Status = status
		worker.LastSeen = time.Now()
		worker.CurrentTestID = currentTestID
		worker.LastProgressMessage = progressMsg
		worker.CompletedRequests = completedReqs
		worker.TotalRequests = totalReqs
		log.Printf("Worker %s status updated to %s (Test: %s, Progress: %d/%d).", workerID, status, currentTestID, completedReqs, totalReqs)
		return nil
	}
	return fmt.Errorf("worker with ID %s not found", workerID)
}

// GetWorkerByID retrieves a worker by its ID from memory.
func (r *InMemoryWorkerRepository) GetWorkerByID(ctx context.Context, workerID string) (*domain.Worker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if worker, ok := r.workers[workerID]; ok {
		return worker, nil
	}
	return nil, fmt.Errorf("worker with ID %s not found", workerID)
}

// GetAvailableWorkers retrieves all workers with 'READY' status from memory.
func (r *InMemoryWorkerRepository) GetAvailableWorkers(ctx context.Context) ([]*domain.Worker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var availableWorkers []*domain.Worker
	for _, worker := range r.workers {
		if worker.Status == "READY" {
			availableWorkers = append(availableWorkers, worker)
		}
	}
	return availableWorkers, nil
}

// GetAllWorkers retrieves all registered workers from memory.
func (r *InMemoryWorkerRepository) GetAllWorkers(ctx context.Context) ([]*domain.Worker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allWorkers []*domain.Worker
	for _, worker := range r.workers {
		allWorkers = append(allWorkers, worker)
	}
	return allWorkers, nil
}

// MarkWorkerOffline marks a worker's status to OFFLINE in memory.
func (r *InMemoryWorkerRepository) MarkWorkerOffline(ctx context.Context, workerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if worker, ok := r.workers[workerID]; ok {
		worker.Status = "OFFLINE"
		worker.LastSeen = time.Now()
		worker.CurrentTestID = "" // Clear current test
		log.Printf("Worker %s marked as OFFLINE.", workerID)
		return nil
	}
	return fmt.Errorf("worker with ID %s not found to mark offline", workerID)
}

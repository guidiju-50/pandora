// Package jobs provides async job management with progress tracking.
package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status represents job status.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Job represents an async processing job.
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      Status                 `json:"status"`
	Progress    int                    `json:"progress"` // 0-100
	Message     string                 `json:"message"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// ProgressUpdate represents a progress update for subscribers.
type ProgressUpdate struct {
	JobID    string `json:"job_id"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
	Status   Status `json:"status"`
}

// Manager handles async job execution and tracking.
type Manager struct {
	jobs        map[string]*Job
	cancelFuncs map[string]context.CancelFunc
	subscribers map[string][]chan ProgressUpdate
	mu          sync.RWMutex
}

// NewManager creates a new job manager.
func NewManager() *Manager {
	return &Manager{
		jobs:        make(map[string]*Job),
		cancelFuncs: make(map[string]context.CancelFunc),
		subscribers: make(map[string][]chan ProgressUpdate),
	}
}

// CreateJob creates a new job and returns its ID.
func (m *Manager) CreateJob(jobType string, input map[string]interface{}) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	job := &Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Status:    StatusPending,
		Progress:  0,
		Message:   "Job created",
		Input:     input,
		Output:    make(map[string]interface{}),
		CreatedAt: time.Now(),
	}

	m.jobs[job.ID] = job
	return job.ID
}

// GetJob returns a job by ID.
func (m *Manager) GetJob(id string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	return job, ok
}

// GetAllJobs returns all jobs.
func (m *Manager) GetAllJobs() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]*Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// StartJob marks a job as running.
func (m *Manager) StartJob(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[id]; ok {
		now := time.Now()
		job.Status = StatusRunning
		job.StartedAt = &now
		job.Message = "Job started"
		m.notifySubscribers(id, ProgressUpdate{
			JobID:    id,
			Progress: 0,
			Message:  "Job started",
			Status:   StatusRunning,
		})
	}
}

// UpdateProgress updates job progress.
func (m *Manager) UpdateProgress(id string, progress int, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[id]; ok {
		job.Progress = progress
		job.Message = message
		m.notifySubscribers(id, ProgressUpdate{
			JobID:    id,
			Progress: progress,
			Message:  message,
			Status:   job.Status,
		})
	}
}

// CompleteJob marks a job as completed with output.
func (m *Manager) CompleteJob(id string, output map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[id]; ok {
		now := time.Now()
		job.Status = StatusCompleted
		job.Progress = 100
		job.Message = "Job completed"
		job.Output = output
		job.CompletedAt = &now
		m.notifySubscribers(id, ProgressUpdate{
			JobID:    id,
			Progress: 100,
			Message:  "Job completed",
			Status:   StatusCompleted,
		})
		m.closeSubscribers(id)
	}
}

// FailJob marks a job as failed with an error.
func (m *Manager) FailJob(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[id]; ok {
		now := time.Now()
		job.Status = StatusFailed
		job.Message = "Job failed"
		job.Error = err.Error()
		job.CompletedAt = &now
		m.notifySubscribers(id, ProgressUpdate{
			JobID:    id,
			Progress: job.Progress,
			Message:  err.Error(),
			Status:   StatusFailed,
		})
		m.closeSubscribers(id)
		delete(m.cancelFuncs, id)
	}
}

// CancelJob cancels a running job.
func (m *Manager) CancelJob(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return false
	}

	// Can only cancel pending or running jobs
	if job.Status != StatusPending && job.Status != StatusRunning {
		return false
	}

	// Call cancel function if exists
	if cancel, ok := m.cancelFuncs[id]; ok {
		cancel()
		delete(m.cancelFuncs, id)
	}

	now := time.Now()
	job.Status = StatusCancelled
	job.Message = "Job cancelled by user"
	job.CompletedAt = &now
	m.notifySubscribers(id, ProgressUpdate{
		JobID:    id,
		Progress: job.Progress,
		Message:  "Job cancelled by user",
		Status:   StatusCancelled,
	})
	m.closeSubscribers(id)

	return true
}

// IsCancelled checks if a job has been cancelled.
func (m *Manager) IsCancelled(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if job, ok := m.jobs[id]; ok {
		return job.Status == StatusCancelled
	}
	return false
}

// Subscribe creates a channel to receive progress updates for a job.
func (m *Manager) Subscribe(id string) <-chan ProgressUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan ProgressUpdate, 10)
	m.subscribers[id] = append(m.subscribers[id], ch)

	// Send current state immediately
	if job, ok := m.jobs[id]; ok {
		ch <- ProgressUpdate{
			JobID:    id,
			Progress: job.Progress,
			Message:  job.Message,
			Status:   job.Status,
		}
	}

	return ch
}

// Unsubscribe removes a subscriber.
func (m *Manager) Unsubscribe(id string, ch <-chan ProgressUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[id]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[id] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}
}

func (m *Manager) notifySubscribers(id string, update ProgressUpdate) {
	for _, ch := range m.subscribers[id] {
		select {
		case ch <- update:
		default:
			// Channel full, skip
		}
	}
}

func (m *Manager) closeSubscribers(id string) {
	for _, ch := range m.subscribers[id] {
		close(ch)
	}
	delete(m.subscribers, id)
}

// RunAsync executes a job function asynchronously with cancellation support.
func (m *Manager) RunAsync(ctx context.Context, jobID string, fn func(ctx context.Context, updateProgress func(int, string)) (map[string]interface{}, error)) {
	// Create cancellable context
	jobCtx, cancel := context.WithCancel(ctx)
	
	// Store cancel function
	m.mu.Lock()
	m.cancelFuncs[jobID] = cancel
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.cancelFuncs, jobID)
			m.mu.Unlock()
		}()

		m.StartJob(jobID)

		updateProgress := func(progress int, message string) {
			// Check if cancelled before updating
			if m.IsCancelled(jobID) {
				return
			}
			m.UpdateProgress(jobID, progress, message)
		}

		result, err := fn(jobCtx, updateProgress)
		
		// Check if job was cancelled
		if m.IsCancelled(jobID) {
			return // Already marked as cancelled
		}
		
		if err != nil {
			// Check if error is due to context cancellation
			if jobCtx.Err() == context.Canceled {
				m.mu.Lock()
				if job, ok := m.jobs[jobID]; ok && job.Status != StatusCancelled {
					now := time.Now()
					job.Status = StatusCancelled
					job.Message = "Job cancelled"
					job.CompletedAt = &now
				}
				m.mu.Unlock()
				return
			}
			m.FailJob(jobID, err)
			return
		}

		m.CompleteJob(jobID, result)
	}()
}

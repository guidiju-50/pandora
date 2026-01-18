// Package repository provides data access layer.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/guidiju-50/pandora/CONTROL/internal/models"
)

// JobRepository handles job data operations.
type JobRepository struct {
	db *sqlx.DB
}

// NewJobRepository creates a new job repository.
func NewJobRepository(db *sqlx.DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create creates a new job.
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	job.ID = uuid.New()
	job.CreatedAt = time.Now()
	job.Status = models.JobStatusPending
	job.Progress = 0

	inputJSON, err := json.Marshal(job.Input)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO jobs (id, project_id, type, status, priority, input, progress, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.db.ExecContext(ctx, query,
		job.ID, job.ProjectID, job.Type, job.Status, job.Priority, inputJSON, job.Progress, job.CreatedBy, job.CreatedAt)
	return err
}

// GetByID retrieves a job by ID.
func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	var job jobRow
	query := `SELECT * FROM jobs WHERE id = $1`
	err := r.db.GetContext(ctx, &job, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return job.toModel()
}

// List retrieves jobs with pagination.
func (r *JobRepository) List(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.Job, error) {
	var rows []jobRow
	query := `SELECT * FROM jobs WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &rows, query, projectID, limit, offset)
	if err != nil {
		return nil, err
	}

	jobs := make([]*models.Job, 0, len(rows))
	for _, row := range rows {
		job, err := row.toModel()
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// ListByStatus retrieves jobs by status.
func (r *JobRepository) ListByStatus(ctx context.Context, status models.JobStatus, limit int) ([]*models.Job, error) {
	var rows []jobRow
	query := `SELECT * FROM jobs WHERE status = $1 ORDER BY priority DESC, created_at ASC LIMIT $2`
	err := r.db.SelectContext(ctx, &rows, query, status, limit)
	if err != nil {
		return nil, err
	}

	jobs := make([]*models.Job, 0, len(rows))
	for _, row := range rows {
		job, err := row.toModel()
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// UpdateStatus updates the status of a job.
func (r *JobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.JobStatus) error {
	query := `UPDATE jobs SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// UpdateProgress updates the progress of a job.
func (r *JobRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress int) error {
	query := `UPDATE jobs SET progress = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, progress, id)
	return err
}

// Start marks a job as started.
func (r *JobRepository) Start(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE jobs SET status = $1, started_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, models.JobStatusRunning, time.Now(), id)
	return err
}

// Complete marks a job as completed.
func (r *JobRepository) Complete(ctx context.Context, id uuid.UUID, output map[string]any) error {
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return err
	}

	query := `UPDATE jobs SET status = $1, output = $2, progress = 100, completed_at = $3 WHERE id = $4`
	_, err = r.db.ExecContext(ctx, query, models.JobStatusCompleted, outputJSON, time.Now(), id)
	return err
}

// Fail marks a job as failed.
func (r *JobRepository) Fail(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `UPDATE jobs SET status = $1, error = $2, completed_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, models.JobStatusFailed, errMsg, time.Now(), id)
	return err
}

// jobRow is a helper struct for database scanning.
type jobRow struct {
	ID          uuid.UUID      `db:"id"`
	ProjectID   uuid.UUID      `db:"project_id"`
	Type        models.JobType `db:"type"`
	Status      models.JobStatus `db:"status"`
	Priority    int            `db:"priority"`
	Input       []byte         `db:"input"`
	Output      []byte         `db:"output"`
	Error       string         `db:"error"`
	Progress    int            `db:"progress"`
	CreatedBy   uuid.UUID      `db:"created_by"`
	CreatedAt   time.Time      `db:"created_at"`
	StartedAt   *time.Time     `db:"started_at"`
	CompletedAt *time.Time     `db:"completed_at"`
}

func (r *jobRow) toModel() (*models.Job, error) {
	job := &models.Job{
		ID:          r.ID,
		ProjectID:   r.ProjectID,
		Type:        r.Type,
		Status:      r.Status,
		Priority:    r.Priority,
		Error:       r.Error,
		Progress:    r.Progress,
		CreatedBy:   r.CreatedBy,
		CreatedAt:   r.CreatedAt,
		StartedAt:   r.StartedAt,
		CompletedAt: r.CompletedAt,
	}

	if len(r.Input) > 0 {
		if err := json.Unmarshal(r.Input, &job.Input); err != nil {
			return nil, err
		}
	}

	if len(r.Output) > 0 {
		if err := json.Unmarshal(r.Output, &job.Output); err != nil {
			return nil, err
		}
	}

	return job, nil
}

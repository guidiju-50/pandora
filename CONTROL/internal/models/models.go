// Package models defines data structures for the CONTROL module.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a system user.
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Name         string     `json:"name" db:"name"`
	Role         Role       `json:"role" db:"role"`
	Active       bool       `json:"active" db:"active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// Role represents user roles.
type Role string

const (
	RoleAdmin      Role = "admin"
	RoleResearcher Role = "researcher"
	RoleViewer     Role = "viewer"
)

// Project represents a research project.
type Project struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	OwnerID     uuid.UUID  `json:"owner_id" db:"owner_id"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Experiment represents an experiment within a project.
type Experiment struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	ProjectID   uuid.UUID         `json:"project_id" db:"project_id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Organism    string            `json:"organism" db:"organism"`
	Platform    string            `json:"platform" db:"platform"`
	Status      string            `json:"status" db:"status"`
	Metadata    map[string]string `json:"metadata" db:"metadata"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// Sample represents a biological sample.
type Sample struct {
	ID           uuid.UUID         `json:"id" db:"id"`
	ExperimentID uuid.UUID         `json:"experiment_id" db:"experiment_id"`
	Name         string            `json:"name" db:"name"`
	Accession    string            `json:"accession" db:"accession"`
	Condition    string            `json:"condition" db:"condition"`
	Replicate    int               `json:"replicate" db:"replicate"`
	Metadata     map[string]string `json:"metadata" db:"metadata"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// Job represents a processing or analysis job.
type Job struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	ProjectID   uuid.UUID         `json:"project_id" db:"project_id"`
	Type        JobType           `json:"type" db:"type"`
	Status      JobStatus         `json:"status" db:"status"`
	Priority    int               `json:"priority" db:"priority"`
	Input       map[string]any    `json:"input" db:"input"`
	Output      map[string]any    `json:"output,omitempty" db:"output"`
	Error       string            `json:"error,omitempty" db:"error"`
	Progress    int               `json:"progress" db:"progress"`
	CreatedBy   uuid.UUID         `json:"created_by" db:"created_by"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
}

// JobType represents the type of job.
type JobType string

const (
	JobTypeScrape     JobType = "scrape"
	JobTypeProcess    JobType = "process"
	JobTypeQuantify   JobType = "quantify"
	JobTypeAnalysis   JobType = "analysis"
	JobTypeEnrichment JobType = "enrichment"
)

// JobStatus represents the status of a job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// JobLog represents a log entry for a job.
type JobLog struct {
	ID        uuid.UUID `json:"id" db:"id"`
	JobID     uuid.UUID `json:"job_id" db:"job_id"`
	Level     string    `json:"level" db:"level"`
	Message   string    `json:"message" db:"message"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

// SRARecord represents an imported SRA record.
type SRARecord struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Accession       string    `json:"accession" db:"accession"`
	Title           string    `json:"title" db:"title"`
	Platform        string    `json:"platform" db:"platform"`
	Instrument      string    `json:"instrument" db:"instrument"`
	LibraryStrategy string    `json:"library_strategy" db:"library_strategy"`
	LibrarySource   string    `json:"library_source" db:"library_source"`
	LibraryLayout   string    `json:"library_layout" db:"library_layout"`
	Organism        string    `json:"organism" db:"organism"`
	TaxID           string    `json:"tax_id" db:"tax_id"`
	BioProject      string    `json:"bio_project" db:"bio_project"`
	BioSample       string    `json:"bio_sample" db:"bio_sample"`
	TotalReads      int64     `json:"total_reads" db:"total_reads"`
	TotalBases      int64     `json:"total_bases" db:"total_bases"`
	AvgLength       int       `json:"avg_length" db:"avg_length"`
	ImportedAt      time.Time `json:"imported_at" db:"imported_at"`
}

// Result represents an analysis result.
type Result struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	ExperimentID uuid.UUID      `json:"experiment_id" db:"experiment_id"`
	JobID        uuid.UUID      `json:"job_id" db:"job_id"`
	Type         string         `json:"type" db:"type"`
	Data         map[string]any `json:"data" db:"data"`
	FilePath     string         `json:"file_path,omitempty" db:"file_path"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
}

// RefreshToken represents a JWT refresh token.
type RefreshToken struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

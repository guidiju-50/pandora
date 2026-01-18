// Package models defines data structures for the PROCESSING module.
package models

import "time"

// Job represents a processing job.
type Job struct {
	ID          string    `json:"id"`
	Type        JobType   `json:"type"`
	Status      JobStatus `json:"status"`
	Input       JobInput  `json:"input"`
	Output      JobOutput `json:"output,omitempty"`
	Error       string    `json:"error,omitempty"`
	Progress    int       `json:"progress"`
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// JobType represents the type of job.
type JobType string

const (
	JobTypeScrape  JobType = "scrape"
	JobTypeProcess JobType = "process"
	JobTypeETL     JobType = "etl"
)

// JobStatus represents the status of a job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// JobInput represents input parameters for a job.
type JobInput struct {
	Accession  string            `json:"accession,omitempty"`
	Database   string            `json:"database,omitempty"`
	InputFiles []string          `json:"input_files,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// JobOutput represents output from a job.
type JobOutput struct {
	Files    []string          `json:"files,omitempty"`
	Metrics  map[string]any    `json:"metrics,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SRARecord represents a record from the SRA database.
type SRARecord struct {
	Accession       string    `json:"accession"`
	Title           string    `json:"title"`
	Platform        string    `json:"platform"`
	Instrument      string    `json:"instrument"`
	LibraryName     string    `json:"library_name"`
	LibraryStrategy string    `json:"library_strategy"`
	LibrarySource   string    `json:"library_source"`
	LibraryLayout   string    `json:"library_layout"`
	Organism        string    `json:"organism"`
	TaxID           string    `json:"tax_id"`
	BioProject      string    `json:"bio_project"`
	BioSample       string    `json:"bio_sample"`
	SampleName      string    `json:"sample_name"`
	ReleaseDate     time.Time `json:"release_date"`
	TotalReads      int64     `json:"total_reads"`
	TotalBases      int64     `json:"total_bases"`
	AvgLength       int       `json:"avg_length"`
}

// FASTQFile represents a FASTQ file with metadata.
type FASTQFile struct {
	Path       string  `json:"path"`
	ReadCount  int64   `json:"read_count"`
	BaseCount  int64   `json:"base_count"`
	AvgQuality float64 `json:"avg_quality"`
	GCContent  float64 `json:"gc_content"`
	Paired     bool    `json:"paired"`
	ReadNumber int     `json:"read_number"` // 1 or 2 for paired-end
}

// TrimmingResult represents the result of Trimmomatic processing.
type TrimmingResult struct {
	InputReads     int64    `json:"input_reads"`
	OutputReads    int64    `json:"output_reads"`
	DroppedReads   int64    `json:"dropped_reads"`
	SurvivalRate   float64  `json:"survival_rate"`
	OutputFiles    []string `json:"output_files"`
	ProcessingTime float64  `json:"processing_time_seconds"`
}

// QualityMetrics represents sequence quality metrics.
type QualityMetrics struct {
	TotalReads      int64   `json:"total_reads"`
	TotalBases      int64   `json:"total_bases"`
	MeanQuality     float64 `json:"mean_quality"`
	MedianQuality   float64 `json:"median_quality"`
	Q20Percentage   float64 `json:"q20_percentage"`
	Q30Percentage   float64 `json:"q30_percentage"`
	GCContent       float64 `json:"gc_content"`
	DuplicationRate float64 `json:"duplication_rate"`
	AdapterContent  float64 `json:"adapter_content"`
}

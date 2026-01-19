// Package pipeline provides orchestration for the complete analysis pipeline.
package pipeline

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/models"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/quantify"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/reference"
	"go.uber.org/zap"
)

// JobStatus represents the status of a pipeline job.
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusRunning    JobStatus = "running"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

// PipelineJob represents a complete pipeline job.
type PipelineJob struct {
	ID           string                 `json:"id"`
	Status       JobStatus              `json:"status"`
	Progress     int                    `json:"progress"`
	Stage        string                 `json:"stage"`
	Message      string                 `json:"message"`
	Input        PipelineInput          `json:"input"`
	Output       *PipelineOutput        `json:"output,omitempty"`
	Error        string                 `json:"error,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// PipelineInput contains input parameters for the pipeline.
type PipelineInput struct {
	Accession    string `json:"accession"`
	Organism     string `json:"organism"`
	// Trimmomatic options
	Leading      int    `json:"leading"`
	Trailing     int    `json:"trailing"`
	SlidingWindow string `json:"sliding_window"`
	MinLen       int    `json:"min_len"`
}

// PipelineOutput contains the results of the pipeline.
type PipelineOutput struct {
	FastqFiles       []string                `json:"fastq_files"`
	TrimmedFiles     []string                `json:"trimmed_files"`
	KallistoDir      string                  `json:"kallisto_dir"`
	MatrixFile       string                  `json:"matrix_file"`
	TotalReads       int64                   `json:"total_reads"`
	MappedReads      int64                   `json:"mapped_reads"`
	MappingRate      float64                 `json:"mapping_rate"`
	TranscriptCount  int                     `json:"transcript_count"`
}

// Orchestrator coordinates the complete pipeline.
type Orchestrator struct {
	processingURL  string
	referenceManager *reference.Manager
	kallisto       *quantify.Kallisto
	matrixGen      *quantify.MatrixGenerator
	jobs           sync.Map
	outputDir      string
	logger         *zap.Logger
}

// NewOrchestrator creates a new pipeline orchestrator.
func NewOrchestrator(
	processingURL string,
	refManager *reference.Manager,
	kallisto *quantify.Kallisto,
	matrixGen *quantify.MatrixGenerator,
	outputDir string,
	logger *zap.Logger,
) *Orchestrator {
	return &Orchestrator{
		processingURL:    processingURL,
		referenceManager: refManager,
		kallisto:         kallisto,
		matrixGen:        matrixGen,
		outputDir:        outputDir,
		logger:           logger,
	}
}

// StartPipeline starts a complete analysis pipeline.
func (o *Orchestrator) StartPipeline(ctx context.Context, input PipelineInput) (string, error) {
	jobID := uuid.New().String()

	job := &PipelineJob{
		ID:        jobID,
		Status:    StatusPending,
		Progress:  0,
		Stage:     "Initializing",
		Message:   "Pipeline job created",
		Input:     input,
		CreatedAt: time.Now(),
	}

	o.jobs.Store(jobID, job)
	o.logger.Info("pipeline job created", zap.String("job_id", jobID), zap.String("accession", input.Accession))

	// Run pipeline asynchronously
	go o.runPipeline(context.Background(), job)

	return jobID, nil
}

// GetJob returns a pipeline job by ID.
func (o *Orchestrator) GetJob(jobID string) (*PipelineJob, bool) {
	if job, ok := o.jobs.Load(jobID); ok {
		return job.(*PipelineJob), true
	}
	return nil, false
}

// ListJobs returns all pipeline jobs.
func (o *Orchestrator) ListJobs() []*PipelineJob {
	var jobs []*PipelineJob
	o.jobs.Range(func(key, value interface{}) bool {
		jobs = append(jobs, value.(*PipelineJob))
		return true
	})
	return jobs
}

// runPipeline executes the complete pipeline.
func (o *Orchestrator) runPipeline(ctx context.Context, job *PipelineJob) {
	startTime := time.Now()
	job.Status = StatusRunning
	job.StartedAt = &startTime
	o.jobs.Store(job.ID, job)

	defer func() {
		if r := recover(); r != nil {
			job.Status = StatusFailed
			job.Error = fmt.Sprintf("pipeline panicked: %v", r)
			now := time.Now()
			job.CompletedAt = &now
			o.jobs.Store(job.ID, job)
		}
	}()

	output := &PipelineOutput{}

	// Stage 1: Ensure reference index (0-20%)
	o.updateProgress(job, 5, "Preparing reference index", "Checking Kallisto index for "+job.Input.Organism)

	indexPath, err := o.ensureIndex(ctx, job)
	if err != nil {
		o.failJob(job, "reference preparation failed: "+err.Error())
		return
	}
	o.updateProgress(job, 20, "Reference ready", "Index available at: "+indexPath)

	// Stage 2: Download & Trim via PROCESSING (20-60%)
	o.updateProgress(job, 25, "Starting download", "Requesting download from PROCESSING module")

	fastqFiles, trimmedFiles, err := o.downloadAndTrim(ctx, job)
	if err != nil {
		o.failJob(job, "download/trim failed: "+err.Error())
		return
	}
	output.FastqFiles = fastqFiles
	output.TrimmedFiles = trimmedFiles
	o.updateProgress(job, 60, "Download & Trim complete", fmt.Sprintf("Trimmed files: %d", len(trimmedFiles)))

	// Stage 3: Kallisto quantification (60-85%)
	o.updateProgress(job, 65, "Starting quantification", "Running Kallisto")

	kallistoDir, quantResult, err := o.runKallisto(ctx, job, indexPath, trimmedFiles)
	if err != nil {
		o.failJob(job, "quantification failed: "+err.Error())
		return
	}
	output.KallistoDir = kallistoDir
	output.TotalReads = quantResult.TotalReads
	output.MappedReads = quantResult.MappedReads
	output.MappingRate = quantResult.MappingRate
	output.TranscriptCount = len(quantResult.Transcripts)
	o.updateProgress(job, 85, "Quantification complete", fmt.Sprintf("Mapped %.1f%% of reads", quantResult.MappingRate*100))

	// Stage 4: Generate TPM matrix (85-100%)
	o.updateProgress(job, 90, "Generating TPM matrix", "Creating output file")

	matrixFile, err := o.generateMatrix(ctx, job, kallistoDir)
	if err != nil {
		o.failJob(job, "matrix generation failed: "+err.Error())
		return
	}
	output.MatrixFile = matrixFile

	// Complete
	job.Status = StatusCompleted
	job.Progress = 100
	job.Stage = "Completed"
	job.Message = "Pipeline completed successfully"
	job.Output = output
	now := time.Now()
	job.CompletedAt = &now
	o.jobs.Store(job.ID, job)

	o.logger.Info("pipeline completed",
		zap.String("job_id", job.ID),
		zap.String("accession", job.Input.Accession),
		zap.String("matrix_file", matrixFile),
		zap.Duration("duration", time.Since(startTime)),
	)
}

// ensureIndex ensures the Kallisto index is available.
func (o *Orchestrator) ensureIndex(ctx context.Context, job *PipelineJob) (string, error) {
	organism := job.Input.Organism
	if organism == "" {
		organism = "helicoverpa_armigera" // Default organism
	}

	// Ensure index is available
	err := o.referenceManager.EnsureIndex(ctx, organism, func(stage string, progress int) {
		// Map 0-100 to 5-20 range
		mappedProgress := 5 + (progress * 15 / 100)
		o.updateProgress(job, mappedProgress, "Preparing index", stage)
	})

	if err != nil {
		return "", err
	}

	return o.referenceManager.GetIndexPath(organism)
}

// downloadAndTrim calls the PROCESSING module to download and trim.
func (o *Orchestrator) downloadAndTrim(ctx context.Context, job *PipelineJob) ([]string, []string, error) {
	// Call PROCESSING API
	url := fmt.Sprintf("%s/api/v1/jobs/full-pipeline", o.processingURL)

	// Build request body
	reqBody := fmt.Sprintf(`{
		"accession": "%s",
		"leading": %d,
		"trailing": %d,
		"sliding_window": "%s",
		"min_len": %d
	}`, job.Input.Accession,
		getOrDefault(job.Input.Leading, 3),
		getOrDefault(job.Input.Trailing, 3),
		getOrDefaultStr(job.Input.SlidingWindow, "4:15"),
		getOrDefault(job.Input.MinLen, 36))

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(reqBody))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("calling PROCESSING: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("PROCESSING returned status: %d", resp.StatusCode)
	}

	// Poll for job completion
	// For now, we'll wait and check the output directory
	accession := job.Input.Accession
	outputDir := filepath.Join(o.outputDir, accession)

	// Wait for files to appear (poll every 5 seconds for up to 30 minutes)
	var fastqFiles, trimmedFiles []string
	maxWait := 30 * time.Minute
	pollInterval := 5 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		// Check for trimmed files
		trimmedDir := filepath.Join(outputDir, "trimmed")
		if files, err := filepath.Glob(filepath.Join(trimmedDir, "*.fastq*")); err == nil && len(files) > 0 {
			trimmedFiles = files
		}

		// Check for original fastq files
		if files, err := filepath.Glob(filepath.Join(outputDir, "*.fastq*")); err == nil && len(files) > 0 {
			fastqFiles = files
		}

		if len(trimmedFiles) > 0 {
			o.logger.Info("trimmed files found", zap.Strings("files", trimmedFiles))
			break
		}

		// Update progress
		elapsed := time.Since(startTime)
		progress := 25 + int(elapsed.Minutes()*2) // Slowly increment
		if progress > 55 {
			progress = 55
		}
		o.updateProgress(job, progress, "Downloading & Trimming", fmt.Sprintf("Waiting for PROCESSING module (%v elapsed)", elapsed.Round(time.Second)))
	}

	if len(trimmedFiles) == 0 {
		return nil, nil, fmt.Errorf("no trimmed files found after waiting")
	}

	return fastqFiles, trimmedFiles, nil
}

// runKallisto runs Kallisto quantification.
func (o *Orchestrator) runKallisto(ctx context.Context, job *PipelineJob, indexPath string, trimmedFiles []string) (string, *models.QuantificationResult, error) {
	accession := job.Input.Accession
	kallistoDir := filepath.Join(o.outputDir, accession, "kallisto")

	// Ensure directory exists
	if err := os.MkdirAll(kallistoDir, 0755); err != nil {
		return "", nil, err
	}

	// Find paired files
	var reads1, reads2 string
	for _, f := range trimmedFiles {
		if strings.Contains(f, "_1") || strings.Contains(f, "_paired_1") || strings.Contains(f, "forward") {
			reads1 = f
		} else if strings.Contains(f, "_2") || strings.Contains(f, "_paired_2") || strings.Contains(f, "reverse") {
			reads2 = f
		}
	}

	if reads1 == "" && len(trimmedFiles) > 0 {
		reads1 = trimmedFiles[0]
	}

	opts := quantify.QuantifyOptions{
		SampleID:  accession,
		Reads1:    reads1,
		Reads2:    reads2,
		Index:     indexPath,
		OutputDir: kallistoDir,
		Bootstrap: 100,
	}

	result, err := o.kallisto.Quantify(ctx, opts)
	if err != nil {
		return "", nil, err
	}

	return kallistoDir, result, nil
}

// generateMatrix generates the TPM matrix file.
func (o *Orchestrator) generateMatrix(ctx context.Context, job *PipelineJob, kallistoDir string) (string, error) {
	accession := job.Input.Accession
	matrixFile := filepath.Join(o.outputDir, accession, fmt.Sprintf("%s_matrix_tpm.txt", accession))

	err := o.matrixGen.GenerateSingleSampleMatrix(accession, kallistoDir, matrixFile)
	if err != nil {
		return "", err
	}

	return matrixFile, nil
}

// updateProgress updates the job progress.
func (o *Orchestrator) updateProgress(job *PipelineJob, progress int, stage, message string) {
	job.Progress = progress
	job.Stage = stage
	job.Message = message
	o.jobs.Store(job.ID, job)
	o.logger.Debug("pipeline progress", zap.String("job_id", job.ID), zap.Int("progress", progress), zap.String("stage", stage))
}

// failJob marks a job as failed.
func (o *Orchestrator) failJob(job *PipelineJob, message string) {
	job.Status = StatusFailed
	job.Error = message
	now := time.Now()
	job.CompletedAt = &now
	o.jobs.Store(job.ID, job)
	o.logger.Error("pipeline failed", zap.String("job_id", job.ID), zap.String("error", message))
}

func getOrDefault(val, def int) int {
	if val <= 0 {
		return def
	}
	return val
}

func getOrDefaultStr(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

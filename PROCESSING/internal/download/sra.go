// Package download provides SRA file download capabilities.
package download

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SRADownloader handles downloading SRA files using SRA Toolkit.
type SRADownloader struct {
	outputDir     string
	tempDir       string
	fasterqDump   string
	prefetch      string
	threads       int
	logger        *zap.Logger
}

// Config holds SRA downloader configuration.
type Config struct {
	OutputDir   string
	TempDir     string
	FasterqDump string // Path to fasterq-dump binary
	Prefetch    string // Path to prefetch binary
	Threads     int
}

// NewSRADownloader creates a new SRA downloader.
func NewSRADownloader(cfg Config, logger *zap.Logger) *SRADownloader {
	fasterqDump := cfg.FasterqDump
	if fasterqDump == "" {
		fasterqDump = "fasterq-dump"
	}

	prefetch := cfg.Prefetch
	if prefetch == "" {
		prefetch = "prefetch"
	}

	threads := cfg.Threads
	if threads <= 0 {
		threads = 4
	}

	return &SRADownloader{
		outputDir:   cfg.OutputDir,
		tempDir:     cfg.TempDir,
		fasterqDump: fasterqDump,
		prefetch:    prefetch,
		threads:     threads,
		logger:      logger,
	}
}

// DownloadResult contains the result of an SRR download.
type DownloadResult struct {
	Accession    string        `json:"accession"`
	Files        []string      `json:"files"`
	OutputDir    string        `json:"output_dir"`
	TotalReads   int64         `json:"total_reads,omitempty"`
	Duration     time.Duration `json:"duration"`
	Status       string        `json:"status"`
	ErrorMessage string        `json:"error,omitempty"`
}

// Download downloads an SRR accession and converts to FASTQ.
func (d *SRADownloader) Download(ctx context.Context, accession string) (*DownloadResult, error) {
	start := time.Now()
	
	d.logger.Info("starting SRR download",
		zap.String("accession", accession),
		zap.String("output_dir", d.outputDir),
	)

	result := &DownloadResult{
		Accession: accession,
		OutputDir: d.outputDir,
		Status:    "started",
	}

	// Validate accession format
	if !strings.HasPrefix(accession, "SRR") && !strings.HasPrefix(accession, "ERR") && !strings.HasPrefix(accession, "DRR") {
		result.Status = "failed"
		result.ErrorMessage = "invalid accession format (must start with SRR, ERR, or DRR)"
		return result, fmt.Errorf("invalid accession: %s", accession)
	}

	// Create output directory
	outputPath := filepath.Join(d.outputDir, accession)
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("failed to create output directory: %v", err)
		return result, err
	}

	// Run fasterq-dump directly (it handles download + conversion)
	args := []string{
		accession,
		"--outdir", outputPath,
		"--threads", fmt.Sprintf("%d", d.threads),
		"--split-files",
		"--progress",
	}

	if d.tempDir != "" {
		args = append(args, "--temp", d.tempDir)
	}

	d.logger.Info("running fasterq-dump",
		zap.String("accession", accession),
		zap.Strings("args", args),
	)

	cmd := exec.CommandContext(ctx, d.fasterqDump, args...)
	cmd.Dir = outputPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		d.logger.Error("fasterq-dump failed",
			zap.String("accession", accession),
			zap.Error(err),
			zap.String("output", string(output)),
		)
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("fasterq-dump failed: %v - %s", err, string(output))
		return result, fmt.Errorf("fasterq-dump failed: %w", err)
	}

	// Find generated FASTQ files
	files, err := filepath.Glob(filepath.Join(outputPath, "*.fastq"))
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("failed to find output files: %v", err)
		return result, err
	}

	if len(files) == 0 {
		// Try .fq extension
		files, _ = filepath.Glob(filepath.Join(outputPath, "*.fq"))
	}

	result.Files = files
	result.Duration = time.Since(start)
	result.Status = "completed"

	d.logger.Info("SRR download completed",
		zap.String("accession", accession),
		zap.Int("files", len(files)),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// DownloadWithPrefetch downloads using prefetch first (for large files).
func (d *SRADownloader) DownloadWithPrefetch(ctx context.Context, accession string) (*DownloadResult, error) {
	start := time.Now()
	
	d.logger.Info("starting SRR download with prefetch",
		zap.String("accession", accession),
	)

	result := &DownloadResult{
		Accession: accession,
		OutputDir: d.outputDir,
		Status:    "started",
	}

	// Create output directory
	outputPath := filepath.Join(d.outputDir, accession)
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("failed to create output directory: %v", err)
		return result, err
	}

	// Step 1: Prefetch the SRA file
	prefetchArgs := []string{
		accession,
		"--output-directory", outputPath,
		"--progress",
	}

	d.logger.Info("running prefetch", zap.Strings("args", prefetchArgs))

	prefetchCmd := exec.CommandContext(ctx, d.prefetch, prefetchArgs...)
	if output, err := prefetchCmd.CombinedOutput(); err != nil {
		d.logger.Error("prefetch failed",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("prefetch failed: %v", err)
		return result, err
	}

	// Step 2: Convert to FASTQ with fasterq-dump
	sraFile := filepath.Join(outputPath, accession, accession+".sra")
	if _, err := os.Stat(sraFile); os.IsNotExist(err) {
		// Try alternate location
		sraFile = filepath.Join(outputPath, accession+".sra")
	}

	fasterqArgs := []string{
		sraFile,
		"--outdir", outputPath,
		"--threads", fmt.Sprintf("%d", d.threads),
		"--split-files",
	}

	if d.tempDir != "" {
		fasterqArgs = append(fasterqArgs, "--temp", d.tempDir)
	}

	d.logger.Info("running fasterq-dump", zap.Strings("args", fasterqArgs))

	fasterqCmd := exec.CommandContext(ctx, d.fasterqDump, fasterqArgs...)
	if output, err := fasterqCmd.CombinedOutput(); err != nil {
		d.logger.Error("fasterq-dump failed",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("fasterq-dump failed: %v", err)
		return result, err
	}

	// Find generated FASTQ files
	files, _ := filepath.Glob(filepath.Join(outputPath, "*.fastq"))
	if len(files) == 0 {
		files, _ = filepath.Glob(filepath.Join(outputPath, "*.fq"))
	}

	// Clean up SRA file to save space
	os.RemoveAll(filepath.Join(outputPath, accession))
	os.Remove(sraFile)

	result.Files = files
	result.Duration = time.Since(start)
	result.Status = "completed"

	d.logger.Info("SRR download with prefetch completed",
		zap.String("accession", accession),
		zap.Int("files", len(files)),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// DownloadMultiple downloads multiple SRR accessions.
func (d *SRADownloader) DownloadMultiple(ctx context.Context, accessions []string) ([]*DownloadResult, error) {
	results := make([]*DownloadResult, 0, len(accessions))

	for _, acc := range accessions {
		result, err := d.Download(ctx, acc)
		if err != nil {
			d.logger.Warn("download failed for accession",
				zap.String("accession", acc),
				zap.Error(err),
			)
		}
		results = append(results, result)
	}

	return results, nil
}

// Package download provides SRA file download capabilities.
package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// SmartDownload tries multiple download strategies.
// 1. First tries fasterq-dump if available
// 2. Falls back to ENA direct download if fasterq-dump fails or not available
func (d *SRADownloader) SmartDownload(ctx context.Context, accession string) (*DownloadResult, error) {
	d.logger.Info("starting smart download",
		zap.String("accession", accession),
	)

	// Check if fasterq-dump is available
	if d.isSRAToolkitAvailable() {
		d.logger.Info("SRA Toolkit available, using fasterq-dump")
		result, err := d.Download(ctx, accession)
		if err == nil {
			return result, nil
		}
		d.logger.Warn("fasterq-dump failed, trying ENA fallback",
			zap.Error(err),
		)
	} else {
		d.logger.Info("SRA Toolkit not available, using ENA direct download")
	}

	// Fallback to ENA direct download
	return d.DownloadFromENA(ctx, accession)
}

// isSRAToolkitAvailable checks if fasterq-dump is available and working.
func (d *SRADownloader) isSRAToolkitAvailable() bool {
	cmd := exec.Command(d.fasterqDump, "--version")
	err := cmd.Run()
	return err == nil
}

// ENAFileInfo contains information about a FASTQ file from ENA.
type ENAFileInfo struct {
	RunAccession string `json:"run_accession"`
	FastqFTP     string `json:"fastq_ftp"`
	FastqMD5     string `json:"fastq_md5"`
	FastqBytes   string `json:"fastq_bytes"`
}

// DownloadFromENA downloads FASTQ files directly from ENA (European Nucleotide Archive).
// This is a fallback when SRA Toolkit is not available (e.g., ARM64 environments).
func (d *SRADownloader) DownloadFromENA(ctx context.Context, accession string) (*DownloadResult, error) {
	start := time.Now()

	d.logger.Info("downloading from ENA",
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

	// Get file URLs from ENA API
	enaAPIURL := fmt.Sprintf(
		"https://www.ebi.ac.uk/ena/portal/api/filereport?accession=%s&result=read_run&fields=run_accession,fastq_ftp,fastq_md5,fastq_bytes&format=json",
		accession,
	)

	d.logger.Debug("querying ENA API", zap.String("url", enaAPIURL))

	req, err := http.NewRequestWithContext(ctx, "GET", enaAPIURL, nil)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		return result, err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("ENA API request failed: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("ENA API returned status %d", resp.StatusCode)
		return result, fmt.Errorf("ENA API error: %d", resp.StatusCode)
	}

	var enaFiles []ENAFileInfo
	if err := json.NewDecoder(resp.Body).Decode(&enaFiles); err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("failed to parse ENA response: %v", err)
		return result, err
	}

	if len(enaFiles) == 0 {
		result.Status = "failed"
		result.ErrorMessage = "no files found in ENA for this accession"
		return result, fmt.Errorf("no files found for %s", accession)
	}

	// Download FASTQ files
	var downloadedFiles []string
	for _, enaFile := range enaFiles {
		if enaFile.FastqFTP == "" {
			continue
		}

		// FTP URLs are semicolon-separated for paired-end
		ftpURLs := strings.Split(enaFile.FastqFTP, ";")
		for _, ftpURL := range ftpURLs {
			if ftpURL == "" {
				continue
			}

			// Convert FTP URL to HTTP
			httpURL := "https://" + strings.TrimPrefix(ftpURL, "ftp://")
			
			filename := filepath.Base(ftpURL)
			outputFile := filepath.Join(outputPath, filename)

			d.logger.Info("downloading file",
				zap.String("url", httpURL),
				zap.String("output", outputFile),
			)

			if err := d.downloadFile(ctx, httpURL, outputFile); err != nil {
				d.logger.Warn("download failed", zap.Error(err))
				continue
			}

			// Decompress if gzipped
			if strings.HasSuffix(outputFile, ".gz") {
				decompressed, err := d.decompressGzip(ctx, outputFile)
				if err != nil {
					d.logger.Warn("decompression failed", zap.Error(err))
					downloadedFiles = append(downloadedFiles, outputFile)
				} else {
					downloadedFiles = append(downloadedFiles, decompressed)
					os.Remove(outputFile) // Remove compressed file
				}
			} else {
				downloadedFiles = append(downloadedFiles, outputFile)
			}
		}
	}

	if len(downloadedFiles) == 0 {
		result.Status = "failed"
		result.ErrorMessage = "no files downloaded successfully"
		return result, fmt.Errorf("no files downloaded for %s", accession)
	}

	result.Files = downloadedFiles
	result.Duration = time.Since(start)
	result.Status = "completed"

	d.logger.Info("ENA download completed",
		zap.String("accession", accession),
		zap.Int("files", len(downloadedFiles)),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// downloadFile downloads a file from URL to the specified path.
func (d *SRADownloader) downloadFile(ctx context.Context, url, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Minute} // Large files may take time
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	d.logger.Debug("file downloaded",
		zap.String("file", outputPath),
		zap.Int64("bytes", written),
	)

	return nil
}

// decompressGzip decompresses a gzip file using pigz (parallel) or gunzip.
func (d *SRADownloader) decompressGzip(ctx context.Context, gzFile string) (string, error) {
	outputFile := strings.TrimSuffix(gzFile, ".gz")

	// Try pigz first (parallel, faster)
	cmd := exec.CommandContext(ctx, "pigz", "-d", "-k", gzFile)
	if err := cmd.Run(); err != nil {
		// Fallback to gunzip
		cmd = exec.CommandContext(ctx, "gunzip", "-k", gzFile)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("decompression failed: %w", err)
		}
	}

	return outputFile, nil
}

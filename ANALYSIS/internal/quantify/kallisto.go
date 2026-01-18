// Package quantify provides RNA-seq quantification tools.
package quantify

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/config"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/models"
	"go.uber.org/zap"
)

// Kallisto provides kallisto quantification functionality.
type Kallisto struct {
	config config.KallistoConfig
	threads int
	logger *zap.Logger
}

// NewKallisto creates a new Kallisto quantifier.
func NewKallisto(cfg config.KallistoConfig, threads int, logger *zap.Logger) *Kallisto {
	return &Kallisto{
		config:  cfg,
		threads: threads,
		logger:  logger,
	}
}

// QuantifyOptions holds options for quantification.
type QuantifyOptions struct {
	SampleID   string
	Reads1     string   // Forward reads or single-end
	Reads2     string   // Reverse reads (empty for single-end)
	Index      string   // Kallisto index file
	OutputDir  string
	Bootstrap  int
	Threads    int
	FragLength float64 // For single-end only
	FragSD     float64 // For single-end only
}

// Quantify runs kallisto quantification.
func (k *Kallisto) Quantify(ctx context.Context, opts QuantifyOptions) (*models.QuantificationResult, error) {
	startTime := time.Now()

	k.logger.Info("starting kallisto quantification",
		zap.String("sample", opts.SampleID),
		zap.String("reads1", opts.Reads1),
	)

	// Validate inputs
	if err := k.validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	// Build command
	args := k.buildArgs(opts)

	// Execute kallisto
	cmd := exec.CommandContext(ctx, k.config.Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		k.logger.Error("kallisto failed",
			zap.String("output", string(output)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("kallisto failed: %w", err)
	}

	k.logger.Debug("kallisto output", zap.String("output", string(output)))

	// Parse results
	result, err := k.parseResults(opts)
	if err != nil {
		return nil, fmt.Errorf("parsing results: %w", err)
	}

	result.ID = uuid.New()
	result.SampleID = opts.SampleID
	result.Tool = "kallisto"
	result.ProcessTime = time.Since(startTime).Seconds()
	result.CreatedAt = time.Now()

	k.logger.Info("kallisto completed",
		zap.String("sample", opts.SampleID),
		zap.Int64("mapped_reads", result.MappedReads),
		zap.Float64("mapping_rate", result.MappingRate),
		zap.Float64("duration", result.ProcessTime),
	)

	return result, nil
}

// validateOptions validates quantification options.
func (k *Kallisto) validateOptions(opts QuantifyOptions) error {
	if opts.Reads1 == "" {
		return fmt.Errorf("reads1 is required")
	}
	if _, err := os.Stat(opts.Reads1); err != nil {
		return fmt.Errorf("reads1 not found: %s", opts.Reads1)
	}
	if opts.Reads2 != "" {
		if _, err := os.Stat(opts.Reads2); err != nil {
			return fmt.Errorf("reads2 not found: %s", opts.Reads2)
		}
	}
	if opts.Index == "" {
		return fmt.Errorf("index is required")
	}
	if _, err := os.Stat(opts.Index); err != nil {
		return fmt.Errorf("index not found: %s", opts.Index)
	}
	return nil
}

// buildArgs builds kallisto command arguments.
func (k *Kallisto) buildArgs(opts QuantifyOptions) []string {
	args := []string{"quant"}

	// Index
	args = append(args, "-i", opts.Index)

	// Output directory
	args = append(args, "-o", opts.OutputDir)

	// Threads
	threads := opts.Threads
	if threads <= 0 {
		threads = k.threads
	}
	args = append(args, "-t", strconv.Itoa(threads))

	// Bootstrap
	bootstrap := opts.Bootstrap
	if bootstrap <= 0 {
		bootstrap = k.config.Bootstrap
	}
	args = append(args, "-b", strconv.Itoa(bootstrap))

	// Single-end specific options
	if opts.Reads2 == "" {
		args = append(args, "--single")
		if opts.FragLength > 0 {
			args = append(args, "-l", fmt.Sprintf("%.1f", opts.FragLength))
		} else {
			args = append(args, "-l", "200") // default
		}
		if opts.FragSD > 0 {
			args = append(args, "-s", fmt.Sprintf("%.1f", opts.FragSD))
		} else {
			args = append(args, "-s", "20") // default
		}
	}

	// Input files
	args = append(args, opts.Reads1)
	if opts.Reads2 != "" {
		args = append(args, opts.Reads2)
	}

	return args
}

// parseResults parses kallisto output files.
func (k *Kallisto) parseResults(opts QuantifyOptions) (*models.QuantificationResult, error) {
	result := &models.QuantificationResult{}

	// Parse run_info.json
	runInfoPath := filepath.Join(opts.OutputDir, "run_info.json")
	if data, err := os.ReadFile(runInfoPath); err == nil {
		var runInfo struct {
			NProcessed int64   `json:"n_processed"`
			NUnique    int64   `json:"n_unique"`
			PAligned   float64 `json:"p_pseudoaligned"`
		}
		if err := json.Unmarshal(data, &runInfo); err == nil {
			result.TotalReads = runInfo.NProcessed
			result.MappedReads = runInfo.NUnique
			result.MappingRate = runInfo.PAligned
		}
	}

	// Parse abundance.tsv
	abundancePath := filepath.Join(opts.OutputDir, "abundance.tsv")
	file, err := os.Open(abundancePath)
	if err != nil {
		return nil, fmt.Errorf("opening abundance file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	
	// Skip header
	if scanner.Scan() {
		// header line
	}

	var transcripts []models.TranscriptCount
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 5 {
			continue
		}

		length, _ := strconv.Atoi(fields[1])
		effLength, _ := strconv.ParseFloat(fields[2], 64)
		estCounts, _ := strconv.ParseFloat(fields[3], 64)
		tpm, _ := strconv.ParseFloat(fields[4], 64)

		transcripts = append(transcripts, models.TranscriptCount{
			TranscriptID: fields[0],
			Length:       length,
			EffLength:    effLength,
			EstCounts:    estCounts,
			TPM:          tpm,
		})
	}

	result.Transcripts = transcripts
	return result, nil
}

// BuildIndex builds a kallisto index.
func (k *Kallisto) BuildIndex(ctx context.Context, fastaFile, indexPath string) error {
	k.logger.Info("building kallisto index",
		zap.String("fasta", fastaFile),
		zap.String("index", indexPath),
	)

	cmd := exec.CommandContext(ctx, k.config.Path, "index", "-i", indexPath, fastaFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		k.logger.Error("kallisto index failed",
			zap.String("output", string(output)),
			zap.Error(err),
		)
		return fmt.Errorf("building index: %w", err)
	}

	k.logger.Info("kallisto index built successfully")
	return nil
}

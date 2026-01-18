// Package quantify provides RNA-seq quantification tools.
package quantify

import (
	"bufio"
	"context"
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

// RSEM provides RSEM quantification functionality.
type RSEM struct {
	config  config.RSEMConfig
	threads int
	logger  *zap.Logger
}

// NewRSEM creates a new RSEM quantifier.
func NewRSEM(cfg config.RSEMConfig, threads int, logger *zap.Logger) *RSEM {
	return &RSEM{
		config:  cfg,
		threads: threads,
		logger:  logger,
	}
}

// RSEMOptions holds options for RSEM quantification.
type RSEMOptions struct {
	SampleID    string
	Reads1      string
	Reads2      string
	Reference   string // RSEM reference prefix
	OutputDir   string
	OutputName  string
	Threads     int
	Paired      bool
	Strandedness string // none, forward, reverse
}

// Quantify runs RSEM quantification.
func (r *RSEM) Quantify(ctx context.Context, opts RSEMOptions) (*models.QuantificationResult, error) {
	startTime := time.Now()

	r.logger.Info("starting RSEM quantification",
		zap.String("sample", opts.SampleID),
		zap.Bool("paired", opts.Paired),
	)

	// Build command
	args := r.buildArgs(opts)

	// Execute rsem-calculate-expression
	cmdPath := filepath.Join(r.config.Path, "rsem-calculate-expression")
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	
	// Set environment for bowtie2
	if r.config.Bowtie2Path != "" {
		cmd.Env = append(os.Environ(), "PATH="+r.config.Bowtie2Path+":"+os.Getenv("PATH"))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		r.logger.Error("RSEM failed",
			zap.String("output", string(output)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("RSEM failed: %w", err)
	}

	// Parse results
	result, err := r.parseResults(opts)
	if err != nil {
		return nil, fmt.Errorf("parsing results: %w", err)
	}

	result.ID = uuid.New()
	result.SampleID = opts.SampleID
	result.Tool = "rsem"
	result.ProcessTime = time.Since(startTime).Seconds()
	result.CreatedAt = time.Now()

	r.logger.Info("RSEM completed",
		zap.String("sample", opts.SampleID),
		zap.Float64("duration", result.ProcessTime),
	)

	return result, nil
}

// buildArgs builds RSEM command arguments.
func (r *RSEM) buildArgs(opts RSEMOptions) []string {
	var args []string

	// Paired-end
	if opts.Paired {
		args = append(args, "--paired-end")
	}

	// Threads
	threads := opts.Threads
	if threads <= 0 {
		threads = r.threads
	}
	args = append(args, "-p", strconv.Itoa(threads))

	// Strandedness
	if opts.Strandedness != "" && opts.Strandedness != "none" {
		args = append(args, "--strandedness", opts.Strandedness)
	}

	// Use bowtie2
	args = append(args, "--bowtie2")

	// Append MAQC
	args = append(args, "--append-names")

	// Input files
	if opts.Paired {
		args = append(args, opts.Reads1, opts.Reads2)
	} else {
		args = append(args, opts.Reads1)
	}

	// Reference
	args = append(args, opts.Reference)

	// Output prefix
	outputPrefix := filepath.Join(opts.OutputDir, opts.OutputName)
	args = append(args, outputPrefix)

	return args
}

// parseResults parses RSEM output files.
func (r *RSEM) parseResults(opts RSEMOptions) (*models.QuantificationResult, error) {
	result := &models.QuantificationResult{}

	// Parse genes.results
	genesPath := filepath.Join(opts.OutputDir, opts.OutputName+".genes.results")
	file, err := os.Open(genesPath)
	if err != nil {
		// Try isoforms.results
		genesPath = filepath.Join(opts.OutputDir, opts.OutputName+".isoforms.results")
		file, err = os.Open(genesPath)
		if err != nil {
			return nil, fmt.Errorf("opening results file: %w", err)
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	
	// Skip header
	if scanner.Scan() {
		// header
	}

	var transcripts []models.TranscriptCount
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 7 {
			continue
		}

		// gene_id, transcript_id(s), length, effective_length, expected_count, TPM, FPKM
		length, _ := strconv.Atoi(fields[2])
		effLength, _ := strconv.ParseFloat(fields[3], 64)
		estCounts, _ := strconv.ParseFloat(fields[4], 64)
		tpm, _ := strconv.ParseFloat(fields[5], 64)
		fpkm, _ := strconv.ParseFloat(fields[6], 64)

		transcripts = append(transcripts, models.TranscriptCount{
			GeneID:    fields[0],
			Length:    length,
			EffLength: effLength,
			EstCounts: estCounts,
			TPM:       tpm,
			FPKM:      fpkm,
		})
	}

	result.Transcripts = transcripts
	return result, nil
}

// PrepareReference prepares RSEM reference from transcriptome.
func (r *RSEM) PrepareReference(ctx context.Context, fastaFile, gtfFile, outputPrefix string) error {
	r.logger.Info("preparing RSEM reference",
		zap.String("fasta", fastaFile),
		zap.String("gtf", gtfFile),
	)

	cmdPath := filepath.Join(r.config.Path, "rsem-prepare-reference")
	args := []string{
		"--gtf", gtfFile,
		"--bowtie2",
		fastaFile,
		outputPrefix,
	}

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	if r.config.Bowtie2Path != "" {
		cmd.Env = append(os.Environ(), "PATH="+r.config.Bowtie2Path+":"+os.Getenv("PATH"))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		r.logger.Error("RSEM prepare-reference failed",
			zap.String("output", string(output)),
			zap.Error(err),
		)
		return fmt.Errorf("preparing reference: %w", err)
	}

	r.logger.Info("RSEM reference prepared successfully")
	return nil
}

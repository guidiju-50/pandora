// Package trimming provides integration with Trimmomatic for sequence trimming.
package trimming

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/guidiju-50/pandora/PROCESSING/internal/config"
	"github.com/guidiju-50/pandora/PROCESSING/internal/models"
	"go.uber.org/zap"
)

// Trimmomatic provides a wrapper for the Trimmomatic tool.
type Trimmomatic struct {
	config config.TrimmoConfig
	logger *zap.Logger
}

// NewTrimmomatic creates a new Trimmomatic wrapper.
func NewTrimmomatic(cfg config.TrimmoConfig, logger *zap.Logger) *Trimmomatic {
	return &Trimmomatic{
		config: cfg,
		logger: logger,
	}
}

// Options holds options for a trimming run.
type Options struct {
	InputFile1  string // Forward reads (or single-end reads)
	InputFile2  string // Reverse reads (empty for single-end)
	OutputDir   string
	Leading     int    // Quality threshold for leading bases
	Trailing    int    // Quality threshold for trailing bases
	SlidingWindow string // Window size:quality threshold
	MinLen      int    // Minimum read length
	Threads     int
	AdapterFile string // Path to adapter file
}

// Result holds the result of a trimming operation.
type Result struct {
	InputReads     int64
	OutputReads    int64
	DroppedReads   int64
	SurvivalRate   float64
	OutputFiles    []string
	LogFile        string
	Duration       time.Duration
}

// Run executes Trimmomatic with the given options.
func (t *Trimmomatic) Run(ctx context.Context, opts Options) (*Result, error) {
	startTime := time.Now()

	// Validate inputs
	if err := t.validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	// Determine if paired-end or single-end
	isPaired := opts.InputFile2 != ""

	// Build command
	args := t.buildArgs(opts, isPaired)

	t.logger.Info("running Trimmomatic",
		zap.Bool("paired", isPaired),
		zap.String("input1", opts.InputFile1),
		zap.String("input2", opts.InputFile2),
	)

	// Execute command
	cmd := exec.CommandContext(ctx, "java", args...)
	
	// Capture stderr for parsing results
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting Trimmomatic: %w", err)
	}

	// Parse output
	result := &Result{}
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		t.logger.Debug("trimmomatic output", zap.String("line", line))
		t.parseOutputLine(line, result, isPaired)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("Trimmomatic failed: %w", err)
	}

	result.Duration = time.Since(startTime)
	result.OutputFiles = t.getOutputFiles(opts, isPaired)

	// Calculate survival rate
	if result.InputReads > 0 {
		result.SurvivalRate = float64(result.OutputReads) / float64(result.InputReads) * 100
	}

	t.logger.Info("Trimmomatic completed",
		zap.Int64("input_reads", result.InputReads),
		zap.Int64("output_reads", result.OutputReads),
		zap.Float64("survival_rate", result.SurvivalRate),
		zap.Duration("duration", result.Duration),
	)

	return result, nil
}

// validateOptions validates trimming options.
func (t *Trimmomatic) validateOptions(opts Options) error {
	if opts.InputFile1 == "" {
		return fmt.Errorf("input file 1 is required")
	}

	if _, err := os.Stat(opts.InputFile1); err != nil {
		return fmt.Errorf("input file 1 not found: %s", opts.InputFile1)
	}

	if opts.InputFile2 != "" {
		if _, err := os.Stat(opts.InputFile2); err != nil {
			return fmt.Errorf("input file 2 not found: %s", opts.InputFile2)
		}
	}

	if t.config.JarPath == "" {
		return fmt.Errorf("Trimmomatic JAR path not configured")
	}

	if _, err := os.Stat(t.config.JarPath); err != nil {
		return fmt.Errorf("Trimmomatic JAR not found: %s", t.config.JarPath)
	}

	return nil
}

// buildArgs builds the command line arguments for Trimmomatic.
func (t *Trimmomatic) buildArgs(opts Options, isPaired bool) []string {
	args := []string{
		"-jar", t.config.JarPath,
	}

	// Add mode
	if isPaired {
		args = append(args, "PE")
	} else {
		args = append(args, "SE")
	}

	// Add threads
	threads := opts.Threads
	if threads <= 0 {
		threads = t.config.Threads
	}
	args = append(args, "-threads", strconv.Itoa(threads))

	// Add input files
	args = append(args, opts.InputFile1)
	if isPaired {
		args = append(args, opts.InputFile2)
	}

	// Add output files
	baseName := strings.TrimSuffix(filepath.Base(opts.InputFile1), filepath.Ext(opts.InputFile1))
	baseName = strings.TrimSuffix(baseName, ".fastq")
	baseName = strings.TrimSuffix(baseName, ".fq")
	baseName = strings.TrimSuffix(baseName, "_1")
	baseName = strings.TrimSuffix(baseName, "_R1")

	if isPaired {
		args = append(args,
			filepath.Join(opts.OutputDir, baseName+"_1_paired.fastq.gz"),
			filepath.Join(opts.OutputDir, baseName+"_1_unpaired.fastq.gz"),
			filepath.Join(opts.OutputDir, baseName+"_2_paired.fastq.gz"),
			filepath.Join(opts.OutputDir, baseName+"_2_unpaired.fastq.gz"),
		)
	} else {
		args = append(args,
			filepath.Join(opts.OutputDir, baseName+"_trimmed.fastq.gz"),
		)
	}

	// Add trimming steps
	args = append(args, t.buildTrimmingSteps(opts)...)

	return args
}

// buildTrimmingSteps builds the trimming step arguments.
func (t *Trimmomatic) buildTrimmingSteps(opts Options) []string {
	var steps []string

	// Adapter trimming
	adapterFile := opts.AdapterFile
	if adapterFile == "" && t.config.AdaptersPath != "" {
		adapterFile = filepath.Join(t.config.AdaptersPath, "TruSeq3-PE-2.fa")
	}
	if adapterFile != "" {
		if _, err := os.Stat(adapterFile); err == nil {
			steps = append(steps, fmt.Sprintf("ILLUMINACLIP:%s:2:30:10", adapterFile))
		}
	}

	// Leading quality
	leading := opts.Leading
	if leading <= 0 {
		leading = t.config.Leading
	}
	if leading > 0 {
		steps = append(steps, fmt.Sprintf("LEADING:%d", leading))
	}

	// Trailing quality
	trailing := opts.Trailing
	if trailing <= 0 {
		trailing = t.config.Trailing
	}
	if trailing > 0 {
		steps = append(steps, fmt.Sprintf("TRAILING:%d", trailing))
	}

	// Sliding window
	slidingWindow := opts.SlidingWindow
	if slidingWindow == "" {
		slidingWindow = t.config.SlidingWindow
	}
	if slidingWindow != "" {
		steps = append(steps, fmt.Sprintf("SLIDINGWINDOW:%s", slidingWindow))
	}

	// Minimum length
	minLen := opts.MinLen
	if minLen <= 0 {
		minLen = t.config.MinLen
	}
	if minLen > 0 {
		steps = append(steps, fmt.Sprintf("MINLEN:%d", minLen))
	}

	return steps
}

// parseOutputLine parses a line of Trimmomatic output.
func (t *Trimmomatic) parseOutputLine(line string, result *Result, isPaired bool) {
	// Parse input reads
	// Example: "Input Read Pairs: 1000000"
	// Example: "Input Reads: 1000000"
	if strings.Contains(line, "Input Read") {
		re := regexp.MustCompile(`Input Read(?:s| Pairs): (\d+)`)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			result.InputReads, _ = strconv.ParseInt(matches[1], 10, 64)
		}
	}

	// Parse surviving reads (paired-end)
	// Example: "Both Surviving: 950000 (95.00%)"
	if strings.Contains(line, "Both Surviving:") {
		re := regexp.MustCompile(`Both Surviving: (\d+)`)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			result.OutputReads, _ = strconv.ParseInt(matches[1], 10, 64)
		}
	}

	// Parse surviving reads (single-end)
	// Example: "Surviving: 950000 (95.00%)"
	if strings.Contains(line, "Surviving:") && !strings.Contains(line, "Both") {
		re := regexp.MustCompile(`Surviving: (\d+)`)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			result.OutputReads, _ = strconv.ParseInt(matches[1], 10, 64)
		}
	}

	// Parse dropped reads
	// Example: "Dropped: 50000 (5.00%)"
	if strings.Contains(line, "Dropped:") {
		re := regexp.MustCompile(`Dropped: (\d+)`)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			result.DroppedReads, _ = strconv.ParseInt(matches[1], 10, 64)
		}
	}
}

// getOutputFiles returns the list of output files.
func (t *Trimmomatic) getOutputFiles(opts Options, isPaired bool) []string {
	baseName := strings.TrimSuffix(filepath.Base(opts.InputFile1), filepath.Ext(opts.InputFile1))
	baseName = strings.TrimSuffix(baseName, ".fastq")
	baseName = strings.TrimSuffix(baseName, ".fq")
	baseName = strings.TrimSuffix(baseName, "_1")
	baseName = strings.TrimSuffix(baseName, "_R1")

	if isPaired {
		return []string{
			filepath.Join(opts.OutputDir, baseName+"_1_paired.fastq.gz"),
			filepath.Join(opts.OutputDir, baseName+"_2_paired.fastq.gz"),
		}
	}

	return []string{
		filepath.Join(opts.OutputDir, baseName+"_trimmed.fastq.gz"),
	}
}

// ToModel converts the Result to a models.TrimmingResult.
func (r *Result) ToModel() *models.TrimmingResult {
	return &models.TrimmingResult{
		InputReads:     r.InputReads,
		OutputReads:    r.OutputReads,
		DroppedReads:   r.DroppedReads,
		SurvivalRate:   r.SurvivalRate,
		OutputFiles:    r.OutputFiles,
		ProcessingTime: r.Duration.Seconds(),
	}
}

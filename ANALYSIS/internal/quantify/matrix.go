// Package quantify provides RNA-seq quantification tools.
package quantify

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// MatrixGenerator generates expression matrices from quantification results.
type MatrixGenerator struct {
	logger *zap.Logger
}

// NewMatrixGenerator creates a new matrix generator.
func NewMatrixGenerator(logger *zap.Logger) *MatrixGenerator {
	return &MatrixGenerator{logger: logger}
}

// TranscriptExpression holds TPM value for a transcript in a sample.
type TranscriptExpression struct {
	TranscriptID string
	TPM          float64
}

// SampleData holds quantification data for a sample.
type SampleData struct {
	SampleID    string
	Expressions map[string]float64 // TranscriptID -> TPM
}

// GenerateTPMMatrix generates a TPM matrix file from multiple Kallisto outputs.
// Each sampleDir should contain an abundance.tsv file from Kallisto.
func (m *MatrixGenerator) GenerateTPMMatrix(sampleDirs map[string]string, outputFile string) error {
	m.logger.Info("generating TPM matrix",
		zap.Int("samples", len(sampleDirs)),
		zap.String("output", outputFile),
	)

	// Load all samples
	samples := make([]*SampleData, 0, len(sampleDirs))
	allTranscripts := make(map[string]bool)

	for sampleID, dir := range sampleDirs {
		data, err := m.loadAbundance(sampleID, dir)
		if err != nil {
			m.logger.Warn("failed to load sample",
				zap.String("sample", sampleID),
				zap.Error(err),
			)
			continue
		}
		samples = append(samples, data)

		// Collect all transcript IDs
		for tid := range data.Expressions {
			allTranscripts[tid] = true
		}
	}

	if len(samples) == 0 {
		return fmt.Errorf("no samples loaded successfully")
	}

	// Sort transcript IDs for consistent output
	transcriptIDs := make([]string, 0, len(allTranscripts))
	for tid := range allTranscripts {
		transcriptIDs = append(transcriptIDs, tid)
	}
	sort.Strings(transcriptIDs)

	// Sort samples by ID
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].SampleID < samples[j].SampleID
	})

	// Write matrix file
	if err := m.writeMatrix(transcriptIDs, samples, outputFile); err != nil {
		return fmt.Errorf("writing matrix: %w", err)
	}

	m.logger.Info("TPM matrix generated",
		zap.Int("transcripts", len(transcriptIDs)),
		zap.Int("samples", len(samples)),
		zap.String("file", outputFile),
	)

	return nil
}

// GenerateSingleSampleMatrix generates a TPM matrix for a single sample.
func (m *MatrixGenerator) GenerateSingleSampleMatrix(sampleID, abundanceDir, outputFile string) error {
	m.logger.Info("generating single sample TPM matrix",
		zap.String("sample", sampleID),
		zap.String("output", outputFile),
	)

	data, err := m.loadAbundance(sampleID, abundanceDir)
	if err != nil {
		return fmt.Errorf("loading abundance: %w", err)
	}

	// Sort transcript IDs
	transcriptIDs := make([]string, 0, len(data.Expressions))
	for tid := range data.Expressions {
		transcriptIDs = append(transcriptIDs, tid)
	}
	sort.Strings(transcriptIDs)

	// Write matrix
	if err := m.writeMatrix(transcriptIDs, []*SampleData{data}, outputFile); err != nil {
		return fmt.Errorf("writing matrix: %w", err)
	}

	m.logger.Info("single sample TPM matrix generated",
		zap.Int("transcripts", len(transcriptIDs)),
		zap.String("file", outputFile),
	)

	return nil
}

// loadAbundance loads abundance.tsv from a Kallisto output directory.
func (m *MatrixGenerator) loadAbundance(sampleID, dir string) (*SampleData, error) {
	abundancePath := filepath.Join(dir, "abundance.tsv")
	
	file, err := os.Open(abundancePath)
	if err != nil {
		return nil, fmt.Errorf("opening abundance file: %w", err)
	}
	defer file.Close()

	data := &SampleData{
		SampleID:    sampleID,
		Expressions: make(map[string]float64),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip header
		if lineNum == 1 {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue
		}

		transcriptID := fields[0]
		tpm, err := strconv.ParseFloat(fields[4], 64)
		if err != nil {
			continue
		}

		data.Expressions[transcriptID] = tpm
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading abundance file: %w", err)
	}

	return data, nil
}

// writeMatrix writes the expression matrix to a file.
func (m *MatrixGenerator) writeMatrix(transcriptIDs []string, samples []*SampleData, outputFile string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header: gene_name \t sample1 \t sample2 \t ...
	header := "gene_name"
	for _, sample := range samples {
		header += "\t" + sample.SampleID
	}
	header += "\t\n" // Extra tab for compatibility with example format
	writer.WriteString(header)

	// Write data rows
	for _, tid := range transcriptIDs {
		row := tid
		for _, sample := range samples {
			tpm := sample.Expressions[tid] // Will be 0 if not found
			row += fmt.Sprintf("\t%g", tpm)
		}
		row += "\t\n"
		writer.WriteString(row)
	}

	return nil
}

// MergeAbundanceFiles merges multiple abundance.tsv files into a single matrix.
func (m *MatrixGenerator) MergeAbundanceFiles(abundanceFiles map[string]string, outputFile string) error {
	m.logger.Info("merging abundance files",
		zap.Int("files", len(abundanceFiles)),
		zap.String("output", outputFile),
	)

	// Convert file paths to directory paths
	sampleDirs := make(map[string]string)
	for sampleID, filePath := range abundanceFiles {
		sampleDirs[sampleID] = filepath.Dir(filePath)
	}

	return m.GenerateTPMMatrix(sampleDirs, outputFile)
}

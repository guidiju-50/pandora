// Package trimming provides sequence trimming functionality.
package trimming

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/guidiju-50/pandora/PROCESSING/internal/models"
	"go.uber.org/zap"
)

// QualityChecker provides quality control analysis for FASTQ files.
type QualityChecker struct {
	logger *zap.Logger
}

// NewQualityChecker creates a new QualityChecker.
func NewQualityChecker(logger *zap.Logger) *QualityChecker {
	return &QualityChecker{
		logger: logger,
	}
}

// AnalyzeFile analyzes quality metrics for a FASTQ file.
func (qc *QualityChecker) AnalyzeFile(filePath string) (*models.QualityMetrics, error) {
	qc.logger.Info("analyzing quality", zap.String("file", filePath))

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Handle gzipped files
	if strings.HasSuffix(filePath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("creating gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	return qc.analyze(reader)
}

// analyze performs the quality analysis on a reader.
func (qc *QualityChecker) analyze(reader io.Reader) (*models.QualityMetrics, error) {
	scanner := bufio.NewScanner(reader)

	var (
		totalReads    int64
		totalBases    int64
		totalQuality  int64
		q20Bases      int64
		q30Bases      int64
		gcBases       int64
		lineCount     int
		qualityScores []int
	)

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// FASTQ format: header, sequence, +, quality
		switch lineCount % 4 {
		case 1: // Header
			totalReads++
		case 2: // Sequence
			totalBases += int64(len(line))
			for _, base := range line {
				if base == 'G' || base == 'C' || base == 'g' || base == 'c' {
					gcBases++
				}
			}
		case 0: // Quality (line 4, which is 0 mod 4)
			for _, q := range line {
				// Phred+33 encoding
				score := int(q) - 33
				qualityScores = append(qualityScores, score)
				totalQuality += int64(score)

				if score >= 20 {
					q20Bases++
				}
				if score >= 30 {
					q30Bases++
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	if totalReads == 0 || totalBases == 0 {
		return nil, fmt.Errorf("empty or invalid FASTQ file")
	}

	metrics := &models.QualityMetrics{
		TotalReads:    totalReads,
		TotalBases:    totalBases,
		MeanQuality:   float64(totalQuality) / float64(totalBases),
		Q20Percentage: float64(q20Bases) / float64(totalBases) * 100,
		Q30Percentage: float64(q30Bases) / float64(totalBases) * 100,
		GCContent:     float64(gcBases) / float64(totalBases) * 100,
	}

	// Calculate median quality
	if len(qualityScores) > 0 {
		metrics.MedianQuality = calculateMedian(qualityScores)
	}

	qc.logger.Info("quality analysis completed",
		zap.Int64("reads", metrics.TotalReads),
		zap.Float64("mean_quality", metrics.MeanQuality),
		zap.Float64("q30_pct", metrics.Q30Percentage),
	)

	return metrics, nil
}

// CompareQuality compares quality metrics before and after trimming.
func (qc *QualityChecker) CompareQuality(before, after *models.QualityMetrics) *QualityComparison {
	return &QualityComparison{
		ReadRetention:      float64(after.TotalReads) / float64(before.TotalReads) * 100,
		BaseRetention:      float64(after.TotalBases) / float64(before.TotalBases) * 100,
		QualityImprovement: after.MeanQuality - before.MeanQuality,
		Q30Improvement:     after.Q30Percentage - before.Q30Percentage,
		Before:             before,
		After:              after,
	}
}

// QualityComparison holds the comparison between before and after quality metrics.
type QualityComparison struct {
	ReadRetention      float64                `json:"read_retention"`
	BaseRetention      float64                `json:"base_retention"`
	QualityImprovement float64                `json:"quality_improvement"`
	Q30Improvement     float64                `json:"q30_improvement"`
	Before             *models.QualityMetrics `json:"before"`
	After              *models.QualityMetrics `json:"after"`
}

// calculateMedian calculates the median of a slice of integers.
func calculateMedian(scores []int) float64 {
	n := len(scores)
	if n == 0 {
		return 0
	}

	// For large datasets, use sampling
	if n > 100000 {
		// Sample every nth element
		step := n / 10000
		sampled := make([]int, 0, 10000)
		for i := 0; i < n; i += step {
			sampled = append(sampled, scores[i])
		}
		scores = sampled
		n = len(scores)
	}

	// Simple selection for median (not fully sorted)
	// For production, use a more efficient algorithm
	sorted := make([]int, len(scores))
	copy(sorted, scores)
	quickSelect(sorted, n/2)

	if n%2 == 0 {
		return float64(sorted[n/2-1]+sorted[n/2]) / 2
	}
	return float64(sorted[n/2])
}

// quickSelect partially sorts to find the kth element.
func quickSelect(arr []int, k int) int {
	if len(arr) == 1 {
		return arr[0]
	}

	pivot := arr[len(arr)/2]
	var less, equal, greater []int

	for _, x := range arr {
		switch {
		case x < pivot:
			less = append(less, x)
		case x == pivot:
			equal = append(equal, x)
		default:
			greater = append(greater, x)
		}
	}

	switch {
	case k < len(less):
		return quickSelect(less, k)
	case k < len(less)+len(equal):
		return pivot
	default:
		return quickSelect(greater, k-len(less)-len(equal))
	}
}

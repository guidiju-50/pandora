// Package stats provides statistical analysis functionality.
package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/config"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/models"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/rbridge"
	"go.uber.org/zap"
)

// DifferentialAnalysis provides differential expression analysis.
type DifferentialAnalysis struct {
	rExecutor *rbridge.Executor
	config    config.AnalysisConfig
	tempDir   string
	logger    *zap.Logger
}

// NewDifferentialAnalysis creates a new differential analysis service.
func NewDifferentialAnalysis(rExec *rbridge.Executor, cfg config.AnalysisConfig, tempDir string, logger *zap.Logger) *DifferentialAnalysis {
	return &DifferentialAnalysis{
		rExecutor: rExec,
		config:    cfg,
		tempDir:   tempDir,
		logger:    logger,
	}
}

// DEOptions holds options for differential expression analysis.
type DEOptions struct {
	ExperimentID   uuid.UUID
	CountsFile     string            // Path to counts matrix CSV
	MetadataFile   string            // Path to sample metadata CSV
	Comparison     string            // e.g., "treated_vs_control"
	Condition1     string            // Treatment group
	Condition2     string            // Control group
	Method         string            // deseq2, edger
	PValueThreshold float64
	Log2FCThreshold float64
	MinCountFilter  int
}

// Run executes differential expression analysis.
func (d *DifferentialAnalysis) Run(ctx context.Context, opts DEOptions) (*models.DifferentialExpressionResult, error) {
	d.logger.Info("starting differential expression analysis",
		zap.String("comparison", opts.Comparison),
		zap.String("method", opts.Method),
	)

	// Set defaults
	if opts.Method == "" {
		opts.Method = "deseq2"
	}
	if opts.PValueThreshold == 0 {
		opts.PValueThreshold = d.config.PValueThreshold
	}
	if opts.Log2FCThreshold == 0 {
		opts.Log2FCThreshold = d.config.Log2FCThreshold
	}
	if opts.MinCountFilter == 0 {
		opts.MinCountFilter = d.config.MinCountFilter
	}

	// Create working directory
	workDir := filepath.Join(d.tempDir, fmt.Sprintf("de_%s", uuid.New().String()[:8]))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("creating work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Prepare R arguments
	args := map[string]interface{}{
		"counts_file":      opts.CountsFile,
		"metadata_file":    opts.MetadataFile,
		"condition1":       opts.Condition1,
		"condition2":       opts.Condition2,
		"method":           opts.Method,
		"pvalue_threshold": opts.PValueThreshold,
		"log2fc_threshold": opts.Log2FCThreshold,
		"min_count":        opts.MinCountFilter,
	}

	// Execute R script
	outputFile := filepath.Join(workDir, "de_results.json")
	result, err := d.rExecutor.Execute(ctx, rbridge.ExecuteOptions{
		Script:     "differential_expression.R",
		Args:       args,
		OutputFile: outputFile,
		WorkDir:    workDir,
	})

	if err != nil {
		return nil, fmt.Errorf("R execution failed: %w", err)
	}

	// Parse results
	deResult, err := d.parseResults(result, opts)
	if err != nil {
		return nil, fmt.Errorf("parsing results: %w", err)
	}

	d.logger.Info("differential expression completed",
		zap.Int("significant_up", deResult.SignificantUp),
		zap.Int("significant_down", deResult.SignificantDown),
		zap.Int("total_tested", deResult.TotalTested),
	)

	return deResult, nil
}

// parseResults parses the R output.
func (d *DifferentialAnalysis) parseResults(result *rbridge.Result, opts DEOptions) (*models.DifferentialExpressionResult, error) {
	if result.Data == nil {
		return nil, fmt.Errorf("no data in R result")
	}

	deResult := &models.DifferentialExpressionResult{
		ID:              uuid.New(),
		ExperimentID:    opts.ExperimentID,
		Comparison:      opts.Comparison,
		Method:          opts.Method,
		PValueThreshold: opts.PValueThreshold,
		Log2FCThreshold: opts.Log2FCThreshold,
		CreatedAt:       time.Now(),
	}

	// Parse genes
	if genesData, ok := result.Data["genes"].([]interface{}); ok {
		for _, geneData := range genesData {
			if geneMap, ok := geneData.(map[string]interface{}); ok {
				gene := models.DEGene{
					GeneID:   getString(geneMap, "gene_id"),
					GeneName: getString(geneMap, "gene_name"),
					BaseMean: getFloat(geneMap, "baseMean"),
					Log2FC:   getFloat(geneMap, "log2FoldChange"),
					PValue:   getFloat(geneMap, "pvalue"),
					PAdj:     getFloat(geneMap, "padj"),
				}

				// Determine significance and direction
				gene.Significant = gene.PAdj < opts.PValueThreshold && 
					(gene.Log2FC > opts.Log2FCThreshold || gene.Log2FC < -opts.Log2FCThreshold)

				if gene.Significant {
					if gene.Log2FC > 0 {
						gene.Direction = "up"
						deResult.SignificantUp++
					} else {
						gene.Direction = "down"
						deResult.SignificantDown++
					}
				} else {
					gene.Direction = "ns"
				}

				deResult.Genes = append(deResult.Genes, gene)
			}
		}
	}

	deResult.TotalTested = len(deResult.Genes)

	// Parse summary if available
	if summary, ok := result.Data["summary"].(map[string]interface{}); ok {
		if up, ok := summary["significant_up"].(float64); ok {
			deResult.SignificantUp = int(up)
		}
		if down, ok := summary["significant_down"].(float64); ok {
			deResult.SignificantDown = int(down)
		}
	}

	return deResult, nil
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

// RunPCA performs PCA analysis.
func (d *DifferentialAnalysis) RunPCA(ctx context.Context, countsFile, metadataFile string, experimentID uuid.UUID) (*models.PCAResult, error) {
	d.logger.Info("starting PCA analysis")

	workDir := filepath.Join(d.tempDir, fmt.Sprintf("pca_%s", uuid.New().String()[:8]))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("creating work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	args := map[string]interface{}{
		"counts_file":   countsFile,
		"metadata_file": metadataFile,
		"n_components":  10,
	}

	outputFile := filepath.Join(workDir, "pca_results.json")
	result, err := d.rExecutor.Execute(ctx, rbridge.ExecuteOptions{
		Script:     "pca_analysis.R",
		Args:       args,
		OutputFile: outputFile,
		WorkDir:    workDir,
	})

	if err != nil {
		return nil, fmt.Errorf("PCA failed: %w", err)
	}

	return d.parsePCAResults(result, experimentID)
}

func (d *DifferentialAnalysis) parsePCAResults(result *rbridge.Result, experimentID uuid.UUID) (*models.PCAResult, error) {
	pcaResult := &models.PCAResult{
		ID:           uuid.New(),
		ExperimentID: experimentID,
		CreatedAt:    time.Now(),
	}

	if result.Data == nil {
		return nil, fmt.Errorf("no PCA data")
	}

	// Parse variance explained
	if variance, ok := result.Data["variance_explained"].([]interface{}); ok {
		for _, v := range variance {
			if vf, ok := v.(float64); ok {
				pcaResult.VarianceRatio = append(pcaResult.VarianceRatio, vf)
			}
		}
	}

	// Parse sample scores
	if scores, ok := result.Data["sample_scores"].([]interface{}); ok {
		for _, scoreData := range scores {
			if scoreMap, ok := scoreData.(map[string]interface{}); ok {
				sample := models.SampleScore{
					SampleID:  getString(scoreMap, "sample"),
					Condition: getString(scoreMap, "condition"),
				}

				if pc, ok := scoreMap["scores"].([]interface{}); ok {
					for _, p := range pc {
						if pf, ok := p.(float64); ok {
							sample.Scores = append(sample.Scores, pf)
						}
					}
				}

				pcaResult.SampleScores = append(pcaResult.SampleScores, sample)
			}
		}
	}

	return pcaResult, nil
}

// RunClustering performs hierarchical clustering.
func (d *DifferentialAnalysis) RunClustering(ctx context.Context, countsFile string, experimentID uuid.UUID, method, distance string) (*models.ClusteringResult, error) {
	d.logger.Info("starting clustering analysis",
		zap.String("method", method),
		zap.String("distance", distance),
	)

	workDir := filepath.Join(d.tempDir, fmt.Sprintf("cluster_%s", uuid.New().String()[:8]))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("creating work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	if method == "" {
		method = "ward.D2"
	}
	if distance == "" {
		distance = "euclidean"
	}

	args := map[string]interface{}{
		"counts_file": countsFile,
		"method":      method,
		"distance":    distance,
	}

	outputFile := filepath.Join(workDir, "clustering_results.json")
	result, err := d.rExecutor.Execute(ctx, rbridge.ExecuteOptions{
		Script:     "clustering.R",
		Args:       args,
		OutputFile: outputFile,
		WorkDir:    workDir,
	})

	if err != nil {
		return nil, fmt.Errorf("clustering failed: %w", err)
	}

	clusterResult := &models.ClusteringResult{
		ID:           uuid.New(),
		ExperimentID: experimentID,
		Method:       method,
		Distance:     distance,
		CreatedAt:    time.Now(),
	}

	// Parse dendrogram from result
	if result.Data != nil {
		if dendro, ok := result.Data["dendrogram"].(map[string]interface{}); ok {
			clusterResult.Dendrogram = &models.Dendrogram{}
			
			if labels, ok := dendro["labels"].([]interface{}); ok {
				for _, l := range labels {
					if ls, ok := l.(string); ok {
						clusterResult.Dendrogram.Labels = append(clusterResult.Dendrogram.Labels, ls)
					}
				}
			}
		}
	}

	return clusterResult, nil
}

// SaveResultsJSON saves results to a JSON file.
func SaveResultsJSON(data interface{}, filepath string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, jsonData, 0644)
}

// Package models defines data structures for the ANALYSIS module.
package models

import (
	"time"

	"github.com/google/uuid"
)

// QuantificationResult represents the result of RNA-seq quantification.
type QuantificationResult struct {
	ID           uuid.UUID         `json:"id"`
	SampleID     string            `json:"sample_id"`
	Tool         string            `json:"tool"` // kallisto, rsem, salmon
	Transcripts  []TranscriptCount `json:"transcripts"`
	TotalReads   int64             `json:"total_reads"`
	MappedReads  int64             `json:"mapped_reads"`
	MappingRate  float64           `json:"mapping_rate"`
	ProcessTime  float64           `json:"process_time_seconds"`
	CreatedAt    time.Time         `json:"created_at"`
}

// TranscriptCount represents counts for a single transcript.
type TranscriptCount struct {
	TranscriptID string  `json:"transcript_id"`
	GeneID       string  `json:"gene_id"`
	GeneName     string  `json:"gene_name"`
	Length       int     `json:"length"`
	EffLength    float64 `json:"eff_length"`
	EstCounts    float64 `json:"est_counts"`
	TPM          float64 `json:"tpm"`
	FPKM         float64 `json:"fpkm,omitempty"`
}

// CountMatrix represents a gene expression count matrix.
type CountMatrix struct {
	Genes   []string    `json:"genes"`
	Samples []string    `json:"samples"`
	Counts  [][]float64 `json:"counts"`
}

// DifferentialExpressionResult represents DESeq2/edgeR results.
type DifferentialExpressionResult struct {
	ID              uuid.UUID       `json:"id"`
	ExperimentID    uuid.UUID       `json:"experiment_id"`
	Comparison      string          `json:"comparison"` // e.g., "treatment_vs_control"
	Method          string          `json:"method"`     // deseq2, edger, limma
	Genes           []DEGene        `json:"genes"`
	SignificantUp   int             `json:"significant_up"`
	SignificantDown int             `json:"significant_down"`
	TotalTested     int             `json:"total_tested"`
	PValueThreshold float64         `json:"pvalue_threshold"`
	Log2FCThreshold float64         `json:"log2fc_threshold"`
	CreatedAt       time.Time       `json:"created_at"`
}

// DEGene represents a differentially expressed gene.
type DEGene struct {
	GeneID      string  `json:"gene_id"`
	GeneName    string  `json:"gene_name"`
	BaseMean    float64 `json:"base_mean"`
	Log2FC      float64 `json:"log2_fold_change"`
	LFCSError   float64 `json:"lfcse,omitempty"`
	Stat        float64 `json:"stat,omitempty"`
	PValue      float64 `json:"pvalue"`
	PAdj        float64 `json:"padj"`
	Significant bool    `json:"significant"`
	Direction   string  `json:"direction"` // up, down, ns
}

// PCAResult represents PCA analysis results.
type PCAResult struct {
	ID             uuid.UUID       `json:"id"`
	ExperimentID   uuid.UUID       `json:"experiment_id"`
	Components     []PCAComponent  `json:"components"`
	SampleScores   []SampleScore   `json:"sample_scores"`
	VarianceRatio  []float64       `json:"variance_ratio"`
	TotalVariance  float64         `json:"total_variance"`
	CreatedAt      time.Time       `json:"created_at"`
}

// PCAComponent represents a principal component.
type PCAComponent struct {
	PC       int       `json:"pc"`
	Variance float64   `json:"variance"`
	Loadings []float64 `json:"loadings,omitempty"`
}

// SampleScore represents sample coordinates in PCA space.
type SampleScore struct {
	SampleID  string    `json:"sample_id"`
	Condition string    `json:"condition"`
	Scores    []float64 `json:"scores"` // PC1, PC2, PC3...
}

// ClusteringResult represents hierarchical clustering results.
type ClusteringResult struct {
	ID           uuid.UUID        `json:"id"`
	ExperimentID uuid.UUID        `json:"experiment_id"`
	Method       string           `json:"method"` // ward, complete, average
	Distance     string           `json:"distance"` // euclidean, correlation
	Dendrogram   *Dendrogram      `json:"dendrogram"`
	Clusters     []Cluster        `json:"clusters"`
	CreatedAt    time.Time        `json:"created_at"`
}

// Dendrogram represents a hierarchical clustering tree.
type Dendrogram struct {
	Merge   [][]int   `json:"merge"`
	Height  []float64 `json:"height"`
	Order   []int     `json:"order"`
	Labels  []string  `json:"labels"`
}

// Cluster represents a cluster of samples/genes.
type Cluster struct {
	ID      int      `json:"id"`
	Members []string `json:"members"`
	Size    int      `json:"size"`
}

// EnrichmentResult represents GO/KEGG enrichment results.
type EnrichmentResult struct {
	ID           uuid.UUID        `json:"id"`
	ExperimentID uuid.UUID        `json:"experiment_id"`
	Type         string           `json:"type"` // GO_BP, GO_MF, GO_CC, KEGG
	Terms        []EnrichedTerm   `json:"terms"`
	GenesInput   int              `json:"genes_input"`
	GeneMapped   int              `json:"genes_mapped"`
	CreatedAt    time.Time        `json:"created_at"`
}

// EnrichedTerm represents an enriched GO/KEGG term.
type EnrichedTerm struct {
	TermID      string   `json:"term_id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	PValue      float64  `json:"pvalue"`
	PAdj        float64  `json:"padj"`
	GeneRatio   string   `json:"gene_ratio"`
	BgRatio     string   `json:"bg_ratio"`
	GeneCount   int      `json:"gene_count"`
	Genes       []string `json:"genes"`
}

// AnalysisJob represents an analysis job request.
type AnalysisJob struct {
	ID           uuid.UUID         `json:"id"`
	Type         AnalysisType      `json:"type"`
	Status       JobStatus         `json:"status"`
	Input        map[string]any    `json:"input"`
	Output       map[string]any    `json:"output,omitempty"`
	Error        string            `json:"error,omitempty"`
	Progress     int               `json:"progress"`
	CreatedAt    time.Time         `json:"created_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// AnalysisType represents the type of analysis.
type AnalysisType string

const (
	AnalysisQuantify      AnalysisType = "quantify"
	AnalysisDifferential  AnalysisType = "differential"
	AnalysisPCA           AnalysisType = "pca"
	AnalysisClustering    AnalysisType = "clustering"
	AnalysisEnrichmentGO  AnalysisType = "enrichment_go"
	AnalysisEnrichmentKEGG AnalysisType = "enrichment_kegg"
)

// JobStatus represents job status.
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

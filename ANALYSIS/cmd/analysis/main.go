// Package main is the entry point for the ANALYSIS module.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/config"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/models"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/pipeline"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/quantify"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/rbridge"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/reference"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/stats"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	flag.Parse()

	logger := initLogger()
	defer logger.Sync()

	logger.Info("starting ANALYSIS module")

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Warn("using default configuration", zap.Error(err))
		cfg = &config.Config{}
	}

	// Initialize components
	rExecutor := rbridge.NewExecutor(cfg.R, logger)
	kallisto := quantify.NewKallisto(cfg.Quantification.Kallisto, cfg.Quantification.Threads, logger)
	rsem := quantify.NewRSEM(cfg.Quantification.RSEM, cfg.Quantification.Threads, logger)
	diffAnalysis := stats.NewDifferentialAnalysis(rExecutor, cfg.Analysis, cfg.Directories.Temp, logger)
	matrixGen := quantify.NewMatrixGenerator(logger)

	// Initialize reference manager for Kallisto indices
	referenceDir := getEnvOrDefault("REFERENCE_DIR", "/data/references")
	kallistoPath := getEnvOrDefault("KALLISTO_PATH", "/opt/kallisto/kallisto")
	refManager := reference.NewManager(referenceDir, kallistoPath, logger)

	// Initialize pipeline orchestrator
	processingURL := getEnvOrDefault("PROCESSING_URL", "http://processing:8081")
	outputDir := getEnvOrDefault("OUTPUT_DIR", "/data/output")
	orchestrator := pipeline.NewOrchestrator(processingURL, refManager, kallisto, matrixGen, outputDir, logger)

	// Setup router
	router := setupRouter(logger, cfg, kallisto, rsem, rExecutor, diffAnalysis, matrixGen, refManager, orchestrator)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("starting HTTP server", zap.Int("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}

func initLogger() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if os.Getenv("ENV") == "development" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}

	return logger
}

func setupRouter(
	logger *zap.Logger,
	cfg *config.Config,
	kallisto *quantify.Kallisto,
	rsem *quantify.RSEM,
	rExecutor *rbridge.Executor,
	diffAnalysis *stats.DifferentialAnalysis,
	matrixGen *quantify.MatrixGenerator,
	refManager *reference.Manager,
	orchestrator *pipeline.Orchestrator,
) *gin.Engine {
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"module":  "ANALYSIS",
			"version": "1.0.0",
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Quantification
		quant := api.Group("/quantify")
		{
			quant.POST("/kallisto", handleKallistoQuant(logger, kallisto, cfg))
			quant.POST("/rsem", handleRSEMQuant(logger, rsem, cfg))
			quant.POST("/matrix", handleGenerateMatrix(logger, matrixGen))
		}

		// Analysis
		analysis := api.Group("/analysis")
		{
			analysis.POST("/differential", handleDifferential(logger, diffAnalysis))
			analysis.POST("/pca", handlePCA(logger, diffAnalysis))
			analysis.POST("/clustering", handleClustering(logger, diffAnalysis))
		}

		// Jobs (internal)
		jobs := api.Group("/jobs")
		{
			jobs.POST("/quantify", handleQuantifyJob(logger, kallisto, rsem, cfg))
			jobs.POST("/differential", handleDifferentialJob(logger, diffAnalysis))
		}

		// Index/Reference management
		refs := api.Group("/references")
		{
			refs.GET("", handleListOrganisms(logger, refManager))
			refs.POST("/ensure", handleEnsureIndex(logger, refManager))
			refs.POST("/custom", handleAddCustomOrganism(logger, refManager))
		}

		// Index management (legacy)
		api.POST("/index/build", handleBuildIndex(logger, kallisto))

		// Complete Pipeline - Download → Trim → Quantify → Matrix
		pipelineGroup := api.Group("/pipeline")
		{
			pipelineGroup.POST("/start", handleStartPipeline(logger, orchestrator))
			pipelineGroup.GET("/jobs", handleListPipelineJobs(logger, orchestrator))
			pipelineGroup.GET("/jobs/:id", handleGetPipelineJob(logger, orchestrator))
			pipelineGroup.GET("/jobs/:id/progress", handlePipelineProgress(logger, orchestrator))
			pipelineGroup.POST("/jobs/:id/cancel", handleCancelPipelineJob(logger, orchestrator))
		}
	}

	return router
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// corsMiddleware handles CORS for cross-origin requests.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Handler functions

type KallistoRequest struct {
	SampleID  string `json:"sample_id" binding:"required"`
	Reads1    string `json:"reads1" binding:"required"`
	Reads2    string `json:"reads2"`
	Index     string `json:"index" binding:"required"`
	OutputDir string `json:"output_dir" binding:"required"`
	Bootstrap int    `json:"bootstrap"`
}

func handleKallistoQuant(logger *zap.Logger, k *quantify.Kallisto, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req KallistoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		opts := quantify.QuantifyOptions{
			SampleID:  req.SampleID,
			Reads1:    req.Reads1,
			Reads2:    req.Reads2,
			Index:     req.Index,
			OutputDir: req.OutputDir,
			Bootstrap: req.Bootstrap,
		}

		result, err := k.Quantify(c.Request.Context(), opts)
		if err != nil {
			logger.Error("kallisto quantification failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func handleRSEMQuant(logger *zap.Logger, r *quantify.RSEM, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SampleID  string `json:"sample_id" binding:"required"`
			Reads1    string `json:"reads1" binding:"required"`
			Reads2    string `json:"reads2"`
			Reference string `json:"reference" binding:"required"`
			OutputDir string `json:"output_dir" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		opts := quantify.RSEMOptions{
			SampleID:   req.SampleID,
			Reads1:     req.Reads1,
			Reads2:     req.Reads2,
			Reference:  req.Reference,
			OutputDir:  req.OutputDir,
			OutputName: req.SampleID,
			Paired:     req.Reads2 != "",
		}

		result, err := r.Quantify(c.Request.Context(), opts)
		if err != nil {
			logger.Error("RSEM quantification failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

type DifferentialRequest struct {
	ExperimentID    string  `json:"experiment_id"`
	CountsFile      string  `json:"counts_file" binding:"required"`
	MetadataFile    string  `json:"metadata_file" binding:"required"`
	Comparison      string  `json:"comparison" binding:"required"`
	Condition1      string  `json:"condition1" binding:"required"`
	Condition2      string  `json:"condition2" binding:"required"`
	Method          string  `json:"method"`
	PValueThreshold float64 `json:"pvalue_threshold"`
	Log2FCThreshold float64 `json:"log2fc_threshold"`
}

func handleDifferential(logger *zap.Logger, da *stats.DifferentialAnalysis) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DifferentialRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		expID := uuid.Nil
		if req.ExperimentID != "" {
			expID, _ = uuid.Parse(req.ExperimentID)
		}

		opts := stats.DEOptions{
			ExperimentID:    expID,
			CountsFile:      req.CountsFile,
			MetadataFile:    req.MetadataFile,
			Comparison:      req.Comparison,
			Condition1:      req.Condition1,
			Condition2:      req.Condition2,
			Method:          req.Method,
			PValueThreshold: req.PValueThreshold,
			Log2FCThreshold: req.Log2FCThreshold,
		}

		result, err := da.Run(c.Request.Context(), opts)
		if err != nil {
			logger.Error("differential analysis failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func handlePCA(logger *zap.Logger, da *stats.DifferentialAnalysis) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ExperimentID string `json:"experiment_id"`
			CountsFile   string `json:"counts_file" binding:"required"`
			MetadataFile string `json:"metadata_file" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		expID := uuid.Nil
		if req.ExperimentID != "" {
			expID, _ = uuid.Parse(req.ExperimentID)
		}

		result, err := da.RunPCA(c.Request.Context(), req.CountsFile, req.MetadataFile, expID)
		if err != nil {
			logger.Error("PCA analysis failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func handleClustering(logger *zap.Logger, da *stats.DifferentialAnalysis) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ExperimentID string `json:"experiment_id"`
			CountsFile   string `json:"counts_file" binding:"required"`
			Method       string `json:"method"`
			Distance     string `json:"distance"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		expID := uuid.Nil
		if req.ExperimentID != "" {
			expID, _ = uuid.Parse(req.ExperimentID)
		}

		result, err := da.RunClustering(c.Request.Context(), req.CountsFile, expID, req.Method, req.Distance)
		if err != nil {
			logger.Error("clustering failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// Job handlers for queue workers

func handleQuantifyJob(logger *zap.Logger, k *quantify.Kallisto, r *quantify.RSEM, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			JobID string         `json:"job_id" binding:"required"`
			Tool  string         `json:"tool" binding:"required"`
			Input map[string]any `json:"input" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Process based on tool
		var result *models.QuantificationResult
		var err error

		switch req.Tool {
		case "kallisto":
			opts := quantify.QuantifyOptions{
				SampleID:  getString(req.Input, "sample_id"),
				Reads1:    getString(req.Input, "reads1"),
				Reads2:    getString(req.Input, "reads2"),
				Index:     getString(req.Input, "index"),
				OutputDir: getString(req.Input, "output_dir"),
			}
			result, err = k.Quantify(c.Request.Context(), opts)
		case "rsem":
			opts := quantify.RSEMOptions{
				SampleID:   getString(req.Input, "sample_id"),
				Reads1:     getString(req.Input, "reads1"),
				Reads2:     getString(req.Input, "reads2"),
				Reference:  getString(req.Input, "reference"),
				OutputDir:  getString(req.Input, "output_dir"),
				OutputName: getString(req.Input, "sample_id"),
				Paired:     getString(req.Input, "reads2") != "",
			}
			result, err = r.Quantify(c.Request.Context(), opts)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown tool"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"job_id": req.JobID,
			"status": "completed",
			"result": result,
		})
	}
}

func handleDifferentialJob(logger *zap.Logger, da *stats.DifferentialAnalysis) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			JobID string         `json:"job_id" binding:"required"`
			Input map[string]any `json:"input" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		opts := stats.DEOptions{
			CountsFile:      getString(req.Input, "counts_file"),
			MetadataFile:    getString(req.Input, "metadata_file"),
			Comparison:      getString(req.Input, "comparison"),
			Condition1:      getString(req.Input, "condition1"),
			Condition2:      getString(req.Input, "condition2"),
			Method:          getString(req.Input, "method"),
			PValueThreshold: getFloat(req.Input, "pvalue_threshold"),
			Log2FCThreshold: getFloat(req.Input, "log2fc_threshold"),
		}

		result, err := da.Run(c.Request.Context(), opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"job_id": req.JobID,
			"status": "completed",
			"result": result,
		})
	}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

// Matrix generation handler

type MatrixRequest struct {
	SampleID     string `json:"sample_id" binding:"required"`
	AbundanceDir string `json:"abundance_dir" binding:"required"`
	OutputFile   string `json:"output_file" binding:"required"`
}

func handleGenerateMatrix(logger *zap.Logger, matrixGen *quantify.MatrixGenerator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MatrixRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := matrixGen.GenerateSingleSampleMatrix(req.SampleID, req.AbundanceDir, req.OutputFile)
		if err != nil {
			logger.Error("matrix generation failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":      "completed",
			"output_file": req.OutputFile,
			"sample_id":   req.SampleID,
		})
	}
}

// Index building handler

type BuildIndexRequest struct {
	FastaFile string `json:"fasta_file" binding:"required"`
	IndexPath string `json:"index_path" binding:"required"`
}

func handleBuildIndex(logger *zap.Logger, kallisto *quantify.Kallisto) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BuildIndexRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := kallisto.BuildIndex(c.Request.Context(), req.FastaFile, req.IndexPath)
		if err != nil {
			logger.Error("index building failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "completed",
			"index_path": req.IndexPath,
		})
	}
}

// Reference management handlers

func handleListOrganisms(logger *zap.Logger, refManager *reference.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		organisms := refManager.ListOrganisms()
		c.JSON(http.StatusOK, gin.H{
			"organisms": organisms,
		})
	}
}

func handleEnsureIndex(logger *zap.Logger, refManager *reference.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Organism string `json:"organism" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := refManager.EnsureIndex(c.Request.Context(), req.Organism, nil)
		if err != nil {
			logger.Error("failed to ensure index", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		indexPath, _ := refManager.GetIndexPath(req.Organism)
		c.JSON(http.StatusOK, gin.H{
			"status":     "completed",
			"organism":   req.Organism,
			"index_path": indexPath,
		})
	}
}

func handleAddCustomOrganism(logger *zap.Logger, refManager *reference.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name           string `json:"name" binding:"required"`
			ScientificName string `json:"scientific_name"`
			TaxID          string `json:"tax_id"`
			IndexPath      string `json:"index_path" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := refManager.AddCustomOrganism(req.Name, req.ScientificName, req.TaxID, req.IndexPath)
		if err != nil {
			logger.Error("failed to add custom organism", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "added",
			"organism": req.Name,
		})
	}
}

// Pipeline handlers

func handleStartPipeline(logger *zap.Logger, orchestrator *pipeline.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Accession     string `json:"accession" binding:"required"`
			Organism      string `json:"organism"`
			Leading       int    `json:"leading"`
			Trailing      int    `json:"trailing"`
			SlidingWindow string `json:"sliding_window"`
			MinLen        int    `json:"min_len"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		input := pipeline.PipelineInput{
			Accession:     req.Accession,
			Organism:      req.Organism,
			Leading:       req.Leading,
			Trailing:      req.Trailing,
			SlidingWindow: req.SlidingWindow,
			MinLen:        req.MinLen,
		}

		jobID, err := orchestrator.StartPipeline(c.Request.Context(), input)
		if err != nil {
			logger.Error("failed to start pipeline", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"status":  "started",
			"job_id":  jobID,
			"message": "Pipeline started. Check /api/v1/pipeline/jobs/" + jobID + " for progress.",
		})
	}
}

func handleListPipelineJobs(logger *zap.Logger, orchestrator *pipeline.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobs := orchestrator.ListJobs()
		c.JSON(http.StatusOK, gin.H{
			"jobs": jobs,
		})
	}
}

func handleGetPipelineJob(logger *zap.Logger, orchestrator *pipeline.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")
		job, found := orchestrator.GetJob(jobID)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusOK, job)
	}
}

func handleCancelPipelineJob(logger *zap.Logger, orchestrator *pipeline.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")
		
		if orchestrator.CancelJob(jobID) {
			logger.Info("pipeline job cancelled", zap.String("job_id", jobID))
			c.JSON(http.StatusOK, gin.H{
				"status":  "cancelled",
				"job_id":  jobID,
				"message": "Pipeline job cancelled successfully",
			})
		} else {
			job, found := orchestrator.GetJob(jobID)
			if !found {
				c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "cannot cancel job",
				"status":  job.Status,
				"message": "Job is already completed, failed, or cancelled",
			})
		}
	}
}

func handlePipelineProgress(logger *zap.Logger, orchestrator *pipeline.Orchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")
		_, found := orchestrator.GetJob(jobID)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		clientGone := c.Request.Context().Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-clientGone:
				return
			case <-ticker.C:
				currentJob, found := orchestrator.GetJob(jobID)
				if !found {
					return
				}

				fmt.Fprintf(c.Writer, "data: {\"progress\":%d,\"stage\":\"%s\",\"message\":\"%s\",\"status\":\"%s\"}\n\n",
					currentJob.Progress, currentJob.Stage, currentJob.Message, currentJob.Status)
				c.Writer.Flush()

				if currentJob.Status == "completed" || currentJob.Status == "failed" {
					return
				}
			}
		}
	}
}

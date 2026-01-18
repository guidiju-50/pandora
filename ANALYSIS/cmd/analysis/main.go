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
	"github.com/guidiju-50/pandora/ANALYSIS/internal/quantify"
	"github.com/guidiju-50/pandora/ANALYSIS/internal/rbridge"
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

	// Setup router
	router := setupRouter(logger, cfg, kallisto, rsem, rExecutor, diffAnalysis)

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
) *gin.Engine {
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

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
	}

	return router
}

// Handler functions

type KallistoRequest struct {
	SampleID   string  `json:"sample_id" binding:"required"`
	Reads1     string  `json:"reads1" binding:"required"`
	Reads2     string  `json:"reads2"`
	Index      string  `json:"index" binding:"required"`
	OutputDir  string  `json:"output_dir" binding:"required"`
	Bootstrap  int     `json:"bootstrap"`
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
			SampleID   string `json:"sample_id" binding:"required"`
			Reads1     string `json:"reads1" binding:"required"`
			Reads2     string `json:"reads2"`
			Reference  string `json:"reference" binding:"required"`
			OutputDir  string `json:"output_dir" binding:"required"`
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
			ExperimentID:   expID,
			CountsFile:     req.CountsFile,
			MetadataFile:   req.MetadataFile,
			Comparison:     req.Comparison,
			Condition1:     req.Condition1,
			Condition2:     req.Condition2,
			Method:         req.Method,
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
			JobID    string         `json:"job_id" binding:"required"`
			Tool     string         `json:"tool" binding:"required"`
			Input    map[string]any `json:"input" binding:"required"`
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

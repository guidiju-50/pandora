// Package main is the entry point for the PROCESSING module.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guidiju-50/pandora/PROCESSING/internal/config"
	"github.com/guidiju-50/pandora/PROCESSING/internal/download"
	"github.com/guidiju-50/pandora/PROCESSING/internal/etl"
	"github.com/guidiju-50/pandora/PROCESSING/internal/jobs"
	"github.com/guidiju-50/pandora/PROCESSING/internal/scraper"
	"github.com/guidiju-50/pandora/PROCESSING/internal/trimming"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	flag.Parse()

	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	logger.Info("starting PROCESSING module")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Warn("using default configuration", zap.Error(err))
		cfg = &config.Config{}
	}

	// Initialize components
	ncbiScraper := scraper.NewNCBIScraper(cfg.Scraper.NCBI, logger)
	loader := etl.NewLoader(cfg.Control, logger)
	pipeline := etl.NewPipeline(cfg.ETL, ncbiScraper, loader, logger)
	trimmomatic := trimming.NewTrimmomatic(cfg.Trimmomatic, logger)
	qualityChecker := trimming.NewQualityChecker(logger)

	// Initialize SRA downloader
	sraDownloader := download.NewSRADownloader(download.Config{
		OutputDir:   getEnvOrDefault("OUTPUT_DIR", "/data/output"),
		TempDir:     getEnvOrDefault("TEMP_DIR", "/tmp/processing"),
		FasterqDump: getEnvOrDefault("FASTERQ_DUMP", "fasterq-dump"),
		Prefetch:    getEnvOrDefault("PREFETCH", "prefetch"),
		Threads:     4,
	}, logger)

	// Initialize job manager
	jobManager := jobs.NewManager()

	// Create HTTP server
	router := setupRouter(logger, pipeline, trimmomatic, qualityChecker, sraDownloader, jobManager)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting HTTP server", zap.Int("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}

// initLogger initializes the zap logger.
func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Check if running in development mode
	if os.Getenv("ENV") == "development" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}

	return logger
}

// getEnvOrDefault gets environment variable or returns default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupRouter configures the HTTP router.
func setupRouter(
	logger *zap.Logger,
	pipeline *etl.Pipeline,
	trimmomatic *trimming.Trimmomatic,
	qualityChecker *trimming.QualityChecker,
	sraDownloader *download.SRADownloader,
	jobManager *jobs.Manager,
) *gin.Engine {
	// Set Gin mode based on environment
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(ginLogger(logger))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"module":  "PROCESSING",
			"version": "1.0.0",
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Job management
		api.GET("/jobs", handleListJobs(jobManager))
		api.GET("/jobs/:id", handleGetJob(jobManager))
		api.GET("/jobs/:id/progress", handleJobProgress(jobManager))

		// Job actions
		jobsGroup := api.Group("/jobs")
		{
			jobsGroup.POST("/scrape", handleScrape(logger, pipeline))
			jobsGroup.POST("/download", handleDownloadAsync(logger, sraDownloader, jobManager))
			jobsGroup.POST("/process", handleProcess(logger, trimmomatic, qualityChecker))
			jobsGroup.POST("/etl", handleETL(logger, pipeline))
			jobsGroup.POST("/full-pipeline", handleFullPipelineAsync(logger, sraDownloader, trimmomatic, qualityChecker, jobManager))
		}

		// Quality check
		api.POST("/quality", handleQualityCheck(logger, qualityChecker))
	}

	return router
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

// ginLogger returns a Gin middleware for logging.
func ginLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

// Handler functions

// ScrapeRequest represents a scraping job request.
type ScrapeRequest struct {
	Query      string   `json:"query" binding:"required"`
	MaxResults int      `json:"max_results"`
	Accessions []string `json:"accessions"`
}

func handleScrape(logger *zap.Logger, pipeline *etl.Pipeline) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ScrapeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()

		var result *etl.ExtractResult
		var err error

		if len(req.Accessions) > 0 {
			result, err = pipeline.ExtractByAccessions(ctx, req.Accessions)
		} else {
			maxResults := req.MaxResults
			if maxResults <= 0 {
				maxResults = 100
			}
			result, err = pipeline.Extract(ctx, req.Query, maxResults)
		}

		if err != nil {
			logger.Error("scrape failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "completed",
			"records": len(result.Records),
			"data":    result.Records,
			"errors":  len(result.Errors),
		})
	}
}

// ProcessRequest represents a processing job request.
type ProcessRequest struct {
	InputFile1    string `json:"input_file_1" binding:"required"`
	InputFile2    string `json:"input_file_2"`
	OutputDir     string `json:"output_dir" binding:"required"`
	Leading       int    `json:"leading"`
	Trailing      int    `json:"trailing"`
	SlidingWindow string `json:"sliding_window"`
	MinLen        int    `json:"min_len"`
}

func handleProcess(logger *zap.Logger, trimmomatic *trimming.Trimmomatic, qc *trimming.QualityChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ProcessRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()

		// Run quality check before
		beforeQuality, err := qc.AnalyzeFile(req.InputFile1)
		if err != nil {
			logger.Warn("pre-quality check failed", zap.Error(err))
		}

		// Run Trimmomatic
		opts := trimming.Options{
			InputFile1:    req.InputFile1,
			InputFile2:    req.InputFile2,
			OutputDir:     req.OutputDir,
			Leading:       req.Leading,
			Trailing:      req.Trailing,
			SlidingWindow: req.SlidingWindow,
			MinLen:        req.MinLen,
		}

		result, err := trimmomatic.Run(ctx, opts)
		if err != nil {
			logger.Error("trimming failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Run quality check after
		var comparison *trimming.QualityComparison
		if beforeQuality != nil && len(result.OutputFiles) > 0 {
			afterQuality, err := qc.AnalyzeFile(result.OutputFiles[0])
			if err == nil {
				comparison = qc.CompareQuality(beforeQuality, afterQuality)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "completed",
			"result":     result.ToModel(),
			"comparison": comparison,
		})
	}
}

// ETLRequest represents an ETL job request.
type ETLRequest struct {
	Query      string `json:"query" binding:"required"`
	MaxResults int    `json:"max_results"`
}

func handleETL(logger *zap.Logger, pipeline *etl.Pipeline) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ETLRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		maxResults := req.MaxResults
		if maxResults <= 0 {
			maxResults = 100
		}

		if err := pipeline.Run(ctx, req.Query, maxResults); err != nil {
			logger.Error("ETL failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "completed",
		})
	}
}

// DownloadRequest represents an SRR download request.
type DownloadRequest struct {
	Accessions  []string `json:"accessions" binding:"required"`
	UsePrefetch bool     `json:"use_prefetch"`
}

func handleDownload(logger *zap.Logger, downloader *download.SRADownloader) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DownloadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		results := make([]*download.DownloadResult, 0, len(req.Accessions))

		for _, acc := range req.Accessions {
			var result *download.DownloadResult
			var err error

			// Use SmartDownload which automatically falls back to ENA if SRA Toolkit is unavailable
			result, err = downloader.SmartDownload(ctx, acc)

			if err != nil {
				logger.Warn("download failed", zap.String("accession", acc), zap.Error(err))
			}
			results = append(results, result)
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "completed",
			"results": results,
		})
	}
}

// FullPipelineRequest represents a full pipeline request (download + process).
type FullPipelineRequest struct {
	Accession     string `json:"accession" binding:"required"`
	UsePrefetch   bool   `json:"use_prefetch"`
	Leading       int    `json:"leading"`
	Trailing      int    `json:"trailing"`
	SlidingWindow string `json:"sliding_window"`
	MinLen        int    `json:"min_len"`
}

func handleFullPipeline(
	logger *zap.Logger,
	downloader *download.SRADownloader,
	trimmomatic *trimming.Trimmomatic,
	qc *trimming.QualityChecker,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req FullPipelineRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		logger.Info("starting full pipeline", zap.String("accession", req.Accession))

		// Step 1: Download (SmartDownload automatically uses best available method)
		downloadResult, err := downloader.SmartDownload(ctx, req.Accession)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("download failed: %v", err),
				"step":  "download",
			})
			return
		}

		if len(downloadResult.Files) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "no FASTQ files generated",
				"step":  "download",
			})
			return
		}

		// Step 2: Quality check before trimming
		beforeQuality, _ := qc.AnalyzeFile(downloadResult.Files[0])

		// Step 3: Trimmomatic processing
		outputDir := downloadResult.OutputDir + "/trimmed"
		opts := trimming.Options{
			InputFile1:    downloadResult.Files[0],
			OutputDir:     outputDir,
			Leading:       req.Leading,
			Trailing:      req.Trailing,
			SlidingWindow: req.SlidingWindow,
			MinLen:        req.MinLen,
		}

		// Check if paired-end
		if len(downloadResult.Files) > 1 {
			opts.InputFile2 = downloadResult.Files[1]
		}

		trimResult, err := trimmomatic.Run(ctx, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":    fmt.Sprintf("trimmomatic failed: %v", err),
				"step":     "trimming",
				"download": downloadResult,
			})
			return
		}

		// Step 4: Quality check after trimming
		var comparison *trimming.QualityComparison
		if beforeQuality != nil && len(trimResult.OutputFiles) > 0 {
			afterQuality, err := qc.AnalyzeFile(trimResult.OutputFiles[0])
			if err == nil {
				comparison = qc.CompareQuality(beforeQuality, afterQuality)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":             "completed",
			"download":           downloadResult,
			"trimming":           trimResult.ToModel(),
			"quality_comparison": comparison,
		})
	}
}

// QualityRequest represents a quality check request.
type QualityRequest struct {
	FilePath string `json:"file_path" binding:"required"`
}

func handleQualityCheck(logger *zap.Logger, qc *trimming.QualityChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req QualityRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		metrics, err := qc.AnalyzeFile(req.FilePath)
		if err != nil {
			logger.Error("quality check failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "completed",
			"metrics": metrics,
		})
	}
}

// Job management handlers

func handleListJobs(jobManager *jobs.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		allJobs := jobManager.GetAllJobs()
		c.JSON(http.StatusOK, gin.H{
			"jobs": allJobs,
		})
	}
}

func handleGetJob(jobManager *jobs.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		job, ok := jobManager.GetJob(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusOK, job)
	}
}

// handleJobProgress returns Server-Sent Events for job progress.
func handleJobProgress(jobManager *jobs.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		job, ok := jobManager.GetJob(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}

		// If job is already completed, just return the final state
		if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusFailed {
			c.JSON(http.StatusOK, gin.H{
				"job_id":   id,
				"progress": job.Progress,
				"message":  job.Message,
				"status":   job.Status,
			})
			return
		}

		// SSE stream for real-time progress
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		ch := jobManager.Subscribe(id)
		defer jobManager.Unsubscribe(id, ch)

		c.Stream(func(w io.Writer) bool {
			select {
			case update, ok := <-ch:
				if !ok {
					return false
				}
				c.SSEvent("progress", update)
				// Stop streaming if job is done
				if update.Status == jobs.StatusCompleted || update.Status == jobs.StatusFailed {
					return false
				}
				return true
			case <-c.Request.Context().Done():
				return false
			}
		})
	}
}

// Async handlers

func handleDownloadAsync(logger *zap.Logger, downloader *download.SRADownloader, jobManager *jobs.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DownloadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if len(req.Accessions) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one accession required"})
			return
		}

		// Create job
		input := map[string]interface{}{
			"accessions":   req.Accessions,
			"use_prefetch": req.UsePrefetch,
		}
		jobID := jobManager.CreateJob("download", input)

		// Run async
		jobManager.RunAsync(context.Background(), jobID, func(ctx context.Context, updateProgress func(int, string)) (map[string]interface{}, error) {
			results := make([]*download.DownloadResult, 0, len(req.Accessions))
			total := len(req.Accessions)

			for i, acc := range req.Accessions {
				progress := (i * 100) / total
				updateProgress(progress, fmt.Sprintf("Downloading %s (%d/%d)...", acc, i+1, total))

				result, err := downloader.SmartDownload(ctx, acc)
				if err != nil {
					logger.Warn("download failed", zap.String("accession", acc), zap.Error(err))
				}
				results = append(results, result)
			}

			return map[string]interface{}{
				"results": results,
			}, nil
		})

		c.JSON(http.StatusAccepted, gin.H{
			"job_id":  jobID,
			"message": "Download job created",
			"status":  "pending",
		})
	}
}

func handleFullPipelineAsync(
	logger *zap.Logger,
	downloader *download.SRADownloader,
	trimmomatic *trimming.Trimmomatic,
	qc *trimming.QualityChecker,
	jobManager *jobs.Manager,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req FullPipelineRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create job
		input := map[string]interface{}{
			"accession":      req.Accession,
			"leading":        req.Leading,
			"trailing":       req.Trailing,
			"sliding_window": req.SlidingWindow,
			"min_len":        req.MinLen,
		}
		jobID := jobManager.CreateJob("full-pipeline", input)

		// Run async
		jobManager.RunAsync(context.Background(), jobID, func(ctx context.Context, updateProgress func(int, string)) (map[string]interface{}, error) {
			logger.Info("starting full pipeline job", zap.String("job_id", jobID), zap.String("accession", req.Accession))

			// Step 1: Download (5-50%) with progress
			updateProgress(5, fmt.Sprintf("Starting download of %s...", req.Accession))

			// Create download progress callback
			downloadProgress := func(progress int, message string) {
				updateProgress(progress, message)
			}

			downloadResult, err := downloader.SmartDownloadWithProgress(ctx, req.Accession, downloadProgress)
			if err != nil {
				return nil, fmt.Errorf("download failed: %w", err)
			}

			if len(downloadResult.Files) == 0 {
				return nil, fmt.Errorf("no FASTQ files generated")
			}

			updateProgress(50, "Download completed, starting quality analysis...")

			// Step 2: Quality check before trimming
			beforeQuality, _ := qc.AnalyzeFile(downloadResult.Files[0])

			updateProgress(55, "Starting Trimmomatic processing...")

			// Step 3: Trimmomatic processing (50-90%)
			outputDir := downloadResult.OutputDir + "/trimmed"
			opts := trimming.Options{
				InputFile1:    downloadResult.Files[0],
				OutputDir:     outputDir,
				Leading:       req.Leading,
				Trailing:      req.Trailing,
				SlidingWindow: req.SlidingWindow,
				MinLen:        req.MinLen,
			}

			if len(downloadResult.Files) > 1 {
				opts.InputFile2 = downloadResult.Files[1]
			}

			trimResult, err := trimmomatic.Run(ctx, opts)
			if err != nil {
				return nil, fmt.Errorf("trimmomatic failed: %w", err)
			}

			updateProgress(90, "Trimming completed, analyzing quality...")

			// Step 4: Quality check after trimming
			var comparison *trimming.QualityComparison
			if beforeQuality != nil && len(trimResult.OutputFiles) > 0 {
				afterQuality, err := qc.AnalyzeFile(trimResult.OutputFiles[0])
				if err == nil {
					comparison = qc.CompareQuality(beforeQuality, afterQuality)
				}
			}

			updateProgress(100, "Pipeline completed successfully")

			return map[string]interface{}{
				"download":           downloadResult,
				"trimming":           trimResult.ToModel(),
				"quality_comparison": comparison,
			}, nil
		})

		c.JSON(http.StatusAccepted, gin.H{
			"job_id":    jobID,
			"message":   "Full pipeline job created",
			"status":    "pending",
			"accession": req.Accession,
		})
	}
}

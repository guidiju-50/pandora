// Package main is the entry point for the PROCESSING module.
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
	"github.com/guidiju-50/pandora/PROCESSING/internal/config"
	"github.com/guidiju-50/pandora/PROCESSING/internal/etl"
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

	// Create HTTP server
	router := setupRouter(logger, pipeline, trimmomatic, qualityChecker)

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

// setupRouter configures the HTTP router.
func setupRouter(
	logger *zap.Logger,
	pipeline *etl.Pipeline,
	trimmomatic *trimming.Trimmomatic,
	qualityChecker *trimming.QualityChecker,
) *gin.Engine {
	// Set Gin mode based on environment
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
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
		// Jobs
		jobs := api.Group("/jobs")
		{
			jobs.POST("/scrape", handleScrape(logger, pipeline))
			jobs.POST("/process", handleProcess(logger, trimmomatic, qualityChecker))
			jobs.POST("/etl", handleETL(logger, pipeline))
		}

		// Quality check
		api.POST("/quality", handleQualityCheck(logger, qualityChecker))
	}

	return router
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

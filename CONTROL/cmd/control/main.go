// Package main is the entry point for the CONTROL module.
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

	"github.com/guidiju-50/pandora/CONTROL/internal/api"
	"github.com/guidiju-50/pandora/CONTROL/internal/auth"
	"github.com/guidiju-50/pandora/CONTROL/internal/config"
	"github.com/guidiju-50/pandora/CONTROL/internal/queue"
	"github.com/guidiju-50/pandora/CONTROL/pkg/database"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	flag.Parse()

	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	logger.Info("starting CONTROL module")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Warn("using default configuration", zap.Error(err))
		cfg = &config.Config{}
	}

	// Connect to PostgreSQL
	db, err := database.NewPostgresDB(cfg.Database, logger)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Connect to RabbitMQ
	rabbitmq := queue.NewRabbitMQ(cfg.RabbitMQ, logger)
	if err := rabbitmq.Connect(); err != nil {
		logger.Warn("failed to connect to RabbitMQ", zap.Error(err))
		// Continue without RabbitMQ - jobs won't be queued
	}
	defer rabbitmq.Close()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWT)

	// Setup router
	router := api.SetupRouter(cfg, db, rabbitmq, jwtManager, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server
	go func() {
		logger.Info("starting HTTP server", zap.Int("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Wait for interrupt
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

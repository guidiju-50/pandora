// Package api provides HTTP API setup.
package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/guidiju-50/pandora/CONTROL/internal/api/handlers"
	"github.com/guidiju-50/pandora/CONTROL/internal/api/middleware"
	"github.com/guidiju-50/pandora/CONTROL/internal/auth"
	"github.com/guidiju-50/pandora/CONTROL/internal/config"
	"github.com/guidiju-50/pandora/CONTROL/internal/models"
	"github.com/guidiju-50/pandora/CONTROL/internal/queue"
	"github.com/guidiju-50/pandora/CONTROL/internal/warehouse/repository"
	"go.uber.org/zap"
)

// SetupRouter configures the HTTP router.
func SetupRouter(
	cfg *config.Config,
	db *sqlx.DB,
	rabbitmq *queue.RabbitMQ,
	jwtManager *auth.JWTManager,
	logger *zap.Logger,
) *gin.Engine {
	// Set Gin mode
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware(cfg.CORS.AllowedOrigins))

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	jobRepo := repository.NewJobRepository(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, jwtManager, logger)
	projectHandler := handlers.NewProjectHandler(projectRepo, logger)
	jobHandler := handlers.NewJobHandler(jobRepo, projectRepo, rabbitmq, logger)
	warehouseHandler := handlers.NewWarehouseHandler(db, logger)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":   "healthy",
			"module":   "CONTROL",
			"version":  "1.0.0",
			"rabbitmq": rabbitmq.IsConnected(),
		})
	})

	// API v1
	api := router.Group("/api/v1")
	{
		// Auth routes (public)
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.Refresh)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(jwtManager))
		{
			// Auth
			protected.GET("/auth/me", authHandler.Me)
			protected.POST("/auth/logout", authHandler.Logout)

			// Projects
			projects := protected.Group("/projects")
			{
				projects.POST("", projectHandler.Create)
				projects.GET("", projectHandler.List)
				projects.GET("/:id", projectHandler.Get)
				projects.PUT("/:id", projectHandler.Update)
				projects.DELETE("/:id", projectHandler.Delete)
			}

			// Jobs
			jobs := protected.Group("/jobs")
			{
				jobs.POST("", jobHandler.Create)
				jobs.GET("", jobHandler.List)
				jobs.GET("/:id", jobHandler.Get)
				jobs.POST("/:id/cancel", jobHandler.Cancel)
			}

			// Warehouse (search)
			warehouse := protected.Group("/warehouse")
			{
				warehouse.GET("/records", warehouseHandler.SearchRecords)
				warehouse.GET("/stats", warehouseHandler.GetStats)
			}

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole(models.RoleAdmin))
			{
				// Add admin-specific routes here
			}
		}

		// Internal API (for other modules)
		internal := api.Group("/internal")
		{
			// Job updates from workers
			internal.POST("/jobs/:id/progress", jobHandler.UpdateProgress)
			internal.POST("/jobs/:id/complete", jobHandler.Complete)
			internal.POST("/jobs/:id/fail", jobHandler.Fail)

			// Warehouse imports
			internal.POST("/warehouse/records", warehouseHandler.ImportRecords)
		}
	}

	return router
}

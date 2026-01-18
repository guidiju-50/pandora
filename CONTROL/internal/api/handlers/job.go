// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guidiju-50/pandora/CONTROL/internal/models"
	"github.com/guidiju-50/pandora/CONTROL/internal/queue"
	"github.com/guidiju-50/pandora/CONTROL/internal/warehouse/repository"
	"go.uber.org/zap"
)

// JobHandler handles job requests.
type JobHandler struct {
	jobRepo     *repository.JobRepository
	projectRepo *repository.ProjectRepository
	rabbitmq    *queue.RabbitMQ
	logger      *zap.Logger
}

// NewJobHandler creates a new job handler.
func NewJobHandler(
	jobRepo *repository.JobRepository,
	projectRepo *repository.ProjectRepository,
	rabbitmq *queue.RabbitMQ,
	logger *zap.Logger,
) *JobHandler {
	return &JobHandler{
		jobRepo:     jobRepo,
		projectRepo: projectRepo,
		rabbitmq:    rabbitmq,
		logger:      logger,
	}
}

// CreateJobRequest represents a job creation request.
type CreateJobRequest struct {
	ProjectID uuid.UUID         `json:"project_id" binding:"required"`
	Type      models.JobType    `json:"type" binding:"required"`
	Priority  int               `json:"priority"`
	Input     map[string]any    `json:"input" binding:"required"`
}

// Create creates a new job.
func (h *JobHandler) Create(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	// Verify project access
	project, err := h.projectRepo.GetByID(c.Request.Context(), req.ProjectID)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	role, _ := c.Get("role")
	if role != models.RoleAdmin && project.OwnerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	job := &models.Job{
		ProjectID: req.ProjectID,
		Type:      req.Type,
		Priority:  req.Priority,
		Input:     req.Input,
		CreatedBy: userID.(uuid.UUID),
	}

	if err := h.jobRepo.Create(c.Request.Context(), job); err != nil {
		h.logger.Error("failed to create job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Publish to appropriate queue
	if err := h.publishJob(c, job); err != nil {
		h.logger.Error("failed to publish job", zap.Error(err))
		// Job is created but not queued - update status
		h.jobRepo.Fail(c.Request.Context(), job.ID, "failed to queue job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue job"})
		return
	}

	// Update status to queued
	h.jobRepo.UpdateStatus(c.Request.Context(), job.ID, models.JobStatusQueued)
	job.Status = models.JobStatusQueued

	c.JSON(http.StatusCreated, job)
}

// publishJob publishes a job to the appropriate queue.
func (h *JobHandler) publishJob(c *gin.Context, job *models.Job) error {
	payload := map[string]any{
		"job_id":     job.ID.String(),
		"project_id": job.ProjectID.String(),
		"type":       string(job.Type),
		"input":      job.Input,
	}

	switch job.Type {
	case models.JobTypeScrape, models.JobTypeProcess:
		return h.rabbitmq.PublishProcessingJob(c.Request.Context(), job.ID.String(), payload)
	case models.JobTypeQuantify, models.JobTypeAnalysis, models.JobTypeEnrichment:
		return h.rabbitmq.PublishAnalysisJob(c.Request.Context(), job.ID.String(), payload)
	default:
		return h.rabbitmq.PublishProcessingJob(c.Request.Context(), job.ID.String(), payload)
	}
}

// List lists jobs for a project.
func (h *JobHandler) List(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	// Verify project access
	project, err := h.projectRepo.GetByID(c.Request.Context(), projectID)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	if role != models.RoleAdmin && project.OwnerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	jobs, err := h.jobRepo.List(c.Request.Context(), projectID, 50, 0)
	if err != nil {
		h.logger.Error("failed to list jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"total": len(jobs),
	})
}

// Get retrieves a job by ID.
func (h *JobHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	job, err := h.jobRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		h.logger.Error("failed to get job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// Cancel cancels a job.
func (h *JobHandler) Cancel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	job, err := h.jobRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Can only cancel pending or queued jobs
	if job.Status != models.JobStatusPending && job.Status != models.JobStatusQueued {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job cannot be cancelled"})
		return
	}

	if err := h.jobRepo.UpdateStatus(c.Request.Context(), id, models.JobStatusCancelled); err != nil {
		h.logger.Error("failed to cancel job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job cancelled"})
}

// UpdateProgress updates job progress (internal API).
func (h *JobHandler) UpdateProgress(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	var req struct {
		Progress int `json:"progress" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.jobRepo.UpdateProgress(c.Request.Context(), id, req.Progress); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "progress updated"})
}

// Complete marks a job as completed (internal API).
func (h *JobHandler) Complete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	var req struct {
		Output map[string]any `json:"output"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.jobRepo.Complete(c.Request.Context(), id, req.Output); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job completed"})
}

// Fail marks a job as failed (internal API).
func (h *JobHandler) Fail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	var req struct {
		Error string `json:"error" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.jobRepo.Fail(c.Request.Context(), id, req.Error); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job marked as failed"})
}

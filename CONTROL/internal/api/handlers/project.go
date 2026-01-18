// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guidiju-50/pandora/CONTROL/internal/models"
	"github.com/guidiju-50/pandora/CONTROL/internal/warehouse/repository"
	"go.uber.org/zap"
)

// ProjectHandler handles project requests.
type ProjectHandler struct {
	projectRepo *repository.ProjectRepository
	logger      *zap.Logger
}

// NewProjectHandler creates a new project handler.
func NewProjectHandler(projectRepo *repository.ProjectRepository, logger *zap.Logger) *ProjectHandler {
	return &ProjectHandler{
		projectRepo: projectRepo,
		logger:      logger,
	}
}

// CreateProjectRequest represents a project creation request.
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// Create creates a new project.
func (h *ProjectHandler) Create(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	project := &models.Project{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID.(uuid.UUID),
	}

	if err := h.projectRepo.Create(c.Request.Context(), project); err != nil {
		h.logger.Error("failed to create project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// List lists projects for the current user.
func (h *ProjectHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	limit := 20
	offset := 0

	var projects []*models.Project
	var err error

	// Admins can see all projects
	if role == models.RoleAdmin {
		projects, err = h.projectRepo.ListAll(c.Request.Context(), limit, offset)
	} else {
		projects, err = h.projectRepo.List(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	}

	if err != nil {
		h.logger.Error("failed to list projects", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"total":    len(projects),
	})
}

// Get retrieves a project by ID.
func (h *ProjectHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	project, err := h.projectRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		h.logger.Error("failed to get project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Check ownership (unless admin)
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	if role != models.RoleAdmin && project.OwnerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// UpdateProjectRequest represents a project update request.
type UpdateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Update updates a project.
func (h *ProjectHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, err := h.projectRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Check ownership
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	if role != models.RoleAdmin && project.OwnerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Update fields
	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Description != "" {
		project.Description = req.Description
	}
	if req.Status != "" {
		project.Status = req.Status
	}

	if err := h.projectRepo.Update(c.Request.Context(), project); err != nil {
		h.logger.Error("failed to update project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// Delete deletes a project.
func (h *ProjectHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	project, err := h.projectRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Check ownership
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	if role != models.RoleAdmin && project.OwnerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.projectRepo.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("failed to delete project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "project deleted"})
}

// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guidiju-50/pandora/CONTROL/internal/auth"
	"github.com/guidiju-50/pandora/CONTROL/internal/models"
	"github.com/guidiju-50/pandora/CONTROL/internal/warehouse/repository"
	"go.uber.org/zap"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	userRepo   *repository.UserRepository
	jwtManager *auth.JWTManager
	logger     *zap.Logger
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(userRepo *repository.UserRepository, jwtManager *auth.JWTManager, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// Register handles user registration.
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check password strength
	if !auth.ValidatePasswordStrength(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must contain uppercase, lowercase, and number"})
		return
	}

	// Check if user exists
	_, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Create user
	user := &models.User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		Name:         req.Name,
		Role:         models.RoleResearcher,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		h.logger.Error("failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Generate tokens
	tokenPair, err := h.jwtManager.GenerateTokenPair(user)
	if err != nil {
		h.logger.Error("failed to generate tokens", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":  user,
		"token": tokenPair,
	})
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login handles user login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user
	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Check if user is active
	if !user.Active {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is disabled"})
		return
	}

	// Update last login
	h.userRepo.UpdateLastLogin(c.Request.Context(), user.ID)

	// Generate tokens
	tokenPair, err := h.jwtManager.GenerateTokenPair(user)
	if err != nil {
		h.logger.Error("failed to generate tokens", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user,
		"token": tokenPair,
	})
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh handles token refresh.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate refresh token
	userID, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Generate new tokens
	tokenPair, err := h.jwtManager.GenerateTokenPair(user)
	if err != nil {
		h.logger.Error("failed to generate tokens", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, tokenPair)
}

// Me returns the current user.
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Logout handles user logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a stateless JWT system, logout is handled client-side
	// Here we could invalidate refresh tokens if stored in DB
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

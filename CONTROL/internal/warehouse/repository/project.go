// Package repository provides data access layer.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/guidiju-50/pandora/CONTROL/internal/models"
)

// ProjectRepository handles project data operations.
type ProjectRepository struct {
	db *sqlx.DB
}

// NewProjectRepository creates a new project repository.
func NewProjectRepository(db *sqlx.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project.
func (r *ProjectRepository) Create(ctx context.Context, project *models.Project) error {
	project.ID = uuid.New()
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	project.Status = "active"

	query := `
		INSERT INTO projects (id, name, description, owner_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		project.ID, project.Name, project.Description, project.OwnerID, project.Status, project.CreatedAt, project.UpdatedAt)
	return err
}

// GetByID retrieves a project by ID.
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	var project models.Project
	query := `SELECT * FROM projects WHERE id = $1`
	err := r.db.GetContext(ctx, &project, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &project, err
}

// List retrieves projects with pagination.
func (r *ProjectRepository) List(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.Project, error) {
	var projects []*models.Project
	query := `SELECT * FROM projects WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &projects, query, ownerID, limit, offset)
	return projects, err
}

// ListAll retrieves all projects with pagination (for admins).
func (r *ProjectRepository) ListAll(ctx context.Context, limit, offset int) ([]*models.Project, error) {
	var projects []*models.Project
	query := `SELECT * FROM projects ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err := r.db.SelectContext(ctx, &projects, query, limit, offset)
	return projects, err
}

// Update updates a project.
func (r *ProjectRepository) Update(ctx context.Context, project *models.Project) error {
	project.UpdatedAt = time.Now()
	query := `
		UPDATE projects SET name = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, project.Name, project.Description, project.Status, project.UpdatedAt, project.ID)
	return err
}

// Delete deletes a project.
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Count returns the total number of projects for a user.
func (r *ProjectRepository) Count(ctx context.Context, ownerID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM projects WHERE owner_id = $1`
	err := r.db.GetContext(ctx, &count, query, ownerID)
	return count, err
}

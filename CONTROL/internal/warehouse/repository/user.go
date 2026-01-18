// Package repository provides data access layer for the warehouse.
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

var ErrNotFound = errors.New("record not found")

// UserRepository handles user data operations.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Active = true

	query := `
		INSERT INTO users (id, email, password_hash, name, role, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Name, user.Role, user.Active, user.CreatedAt, user.UpdatedAt)
	return err
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}

// Update updates a user.
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	query := `
		UPDATE users SET name = $1, role = $2, active = $3, updated_at = $4
		WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, user.Name, user.Role, user.Active, user.UpdatedAt, user.ID)
	return err
}

// UpdateLastLogin updates the last login timestamp.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// List retrieves all users with pagination.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	query := `SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	return users, err
}

// Delete deletes a user.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

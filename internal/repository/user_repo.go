package repository

import (
	"context"
	"errors"
	"fmt"

	"expense_tracker/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository defines operations for user data
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	FindByID(ctx context.Context, id int) (*model.User, error)
}

type userRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	sql := `INSERT INTO users (phone, password_hash, role, created_at) 
            VALUES ($1, $2, $3, $4) RETURNING id`
	err := r.db.QueryRow(ctx, sql, user.Phone, user.PasswordHash, user.Role, user.CreatedAt).Scan(&user.ID)
	if err != nil {
		// TODO: Check for unique constraint violation specifically pgerrcode.UniqueViolation
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// FindByPhone retrieves a user by their phone number
func (r *userRepository) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	user := &model.User{}
	sql := `SELECT id, phone, password_hash, role, created_at FROM users WHERE phone = $1`
	err := r.db.QueryRow(ctx, sql, phone).Scan(&user.ID, &user.Phone, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // User not found is not an error for this method's contract, service layer handles it
		}
		return nil, fmt.Errorf("failed to find user by phone: %w", err)
	}
	return user, nil
}

// FindByID retrieves a user by their ID
func (r *userRepository) FindByID(ctx context.Context, id int) (*model.User, error) {
	user := &model.User{}
	sql := `SELECT id, phone, password_hash, role, created_at FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, sql, id).Scan(&user.ID, &user.Phone, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return user, nil
}

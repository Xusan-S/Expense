package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"expense_tracker/internal/model"
	"expense_tracker/internal/repository"
	"expense_tracker/internal/utils"

	"github.com/jackc/pgx/v5"
)

var (
	ErrUserAlreadyExists  = errors.New("user with this phone number already exists")
	ErrUserNotFound       = errors.New("user not found") // Though Login groups this with InvalidCredentials
	ErrInvalidCredentials = errors.New("invalid phone or password")
)

// AuthService provides authentication related services
type AuthService interface {
	Register(ctx context.Context, phone, password string) (*model.User, string, error)
	Login(ctx context.Context, phone, password string) (*model.User, string, error)
}

type authService struct {
	userRepo repository.UserRepository
	jwtUtil  *utils.JWTUtil
}

// NewAuthService creates a new AuthService
func NewAuthService(userRepo repository.UserRepository, jwtUtil *utils.JWTUtil) AuthService {
	return &authService{
		userRepo: userRepo,
		jwtUtil:  jwtUtil,
	}
}

// Register creates a new user account
func (s *authService) Register(ctx context.Context, phone, password string) (*model.User, string, error) {
	existingUser, err := s.userRepo.FindByPhone(ctx, phone)
	// We expect pgx.ErrNoRows if the user does not exist, which is not an error in this context.
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, "", fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, "", ErrUserAlreadyExists
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	userRole := model.RoleUser // Default role

	// Check for initial admin setup via environment variable
	initialAdminPhone := os.Getenv("INITIAL_ADMIN_PHONE")
	if initialAdminPhone != "" && phone == initialAdminPhone {

		userRole = model.RoleAdmin
		log.Printf("INFO: User %s is being registered as ADMIN via INITIAL_ADMIN_PHONE.", phone)
	}

	user := &model.User{
		Phone:        phone,
		PasswordHash: hashedPassword,
		Role:         userRole, // Set role based on logic above
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("failed to create user in repository: %w", err)
	}

	token, err := s.jwtUtil.GenerateToken(user.ID, user.Role)
	if err != nil {
		log.Printf("ERROR: User %s (ID: %d) created, but failed to generate token: %v", user.Phone, user.ID, err)
		return user, "", fmt.Errorf("user created, but failed to generate token: %w", err)
	}

	return user, token, nil
}

// Login authenticates a user and returns a JWT token
func (s *authService) Login(ctx context.Context, phone, password string) (*model.User, string, error) {
	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) { // Handle actual DB errors
		return nil, "", fmt.Errorf("error finding user by phone: %w", err)
	}
	if user == nil { // This covers pgx.ErrNoRows or if FindByPhone returns nil for not found
		return nil, "", ErrInvalidCredentials // User not found
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", ErrInvalidCredentials // Password mismatch
	}

	token, err := s.jwtUtil.GenerateToken(user.ID, user.Role)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}

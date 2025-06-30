package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
)

// UserUsecase implements domain.UserUsecase
type UserUsecase struct {
	userRepo  domain.UserRepository
	jwtSecret string
}

// NewUserUsecase creates a new user usecase
func NewUserUsecase(userRepo domain.UserRepository, jwtSecret string) *UserUsecase {
	return &UserUsecase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// Login authenticates a user and returns a JWT token
func (uc *UserUsecase) Login(ctx context.Context, username, password string) (*domain.User, string, error) {
	// Get user by username
	user, err := uc.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, "", fmt.Errorf("user account is disabled")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	// Update last login
	uc.userRepo.UpdateLastLogin(ctx, user.ID)

	// Generate JWT token
	token, err := uc.generateJWT(user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Don't return password hash
	user.Password = ""

	return user, token, nil
}

// AuthenticateUser authenticates a user and returns an auth response
func (uc *UserUsecase) AuthenticateUser(ctx context.Context, username, password string) (*domain.AuthResponse, error) {
	user, token, err := uc.Login(ctx, username, password)
	if err != nil {
		return nil, err
	}

	profile := &domain.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Role:        user.Role,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
	}

	return &domain.AuthResponse{
		Token:     token,
		User:      profile,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// ValidateToken validates a JWT token and returns the user
func (uc *UserUsecase) ValidateToken(ctx context.Context, tokenString string) (*domain.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(uc.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}

		user, err := uc.userRepo.GetUserByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user not found")
		}

		if !user.IsActive {
			return nil, fmt.Errorf("user account is disabled")
		}

		// Don't return password hash
		user.Password = ""
		return user, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateJWTToken validates a JWT token and returns the user profile
func (uc *UserUsecase) ValidateJWTToken(ctx context.Context, tokenString string) (*domain.UserProfile, error) {
	user, err := uc.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	return &domain.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Role:        user.Role,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
	}, nil
}

// CreateUser creates a new user
func (uc *UserUsecase) CreateUser(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error) {
	// Check if username already exists
	_, err := uc.userRepo.GetUserByUsername(ctx, req.Username)
	if err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Check if email already exists
	_, err = uc.userRepo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, fmt.Errorf("email already exists")
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set default role if not specified
	role := req.Role
	if role == "" {
		role = "user"
	}

	// Create user
	user := &domain.User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(passwordHash),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      role,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = uc.userRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Don't return password hash
	user.Password = ""
	return user, nil
}

// GetUserProfile gets user profile information
func (uc *UserUsecase) GetUserProfile(ctx context.Context, userID string) (*domain.UserProfile, error) {
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &domain.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Role:        user.Role,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
	}, nil
}

// UpdateUserProfile updates user profile information
func (uc *UserUsecase) UpdateUserProfile(ctx context.Context, userID string, req *domain.UpdateUserRequest) (*domain.UserProfile, error) {
	// Check if email already exists (if being updated)
	if req.Email != "" {
		existingUser, err := uc.userRepo.GetUserByEmail(ctx, req.Email)
		if err == nil && existingUser.ID != userID {
			return nil, fmt.Errorf("email already exists")
		}
	}

	updatedUser, err := uc.userRepo.UpdateUser(ctx, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &domain.UserProfile{
		ID:          updatedUser.ID,
		Username:    updatedUser.Username,
		Email:       updatedUser.Email,
		FirstName:   updatedUser.FirstName,
		LastName:    updatedUser.LastName,
		Role:        updatedUser.Role,
		IsActive:    updatedUser.IsActive,
		CreatedAt:   updatedUser.CreatedAt,
		UpdatedAt:   updatedUser.UpdatedAt,
		LastLoginAt: updatedUser.LastLoginAt,
	}, nil
}

// ChangePassword changes user password
func (uc *UserUsecase) ChangePassword(ctx context.Context, userID string, req *domain.ChangePasswordRequest) error {
	// Get current user
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword))
	if err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	err = uc.userRepo.UpdateUserPassword(ctx, userID, string(newPasswordHash))
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetAllUsers gets all users (admin only)
func (uc *UserUsecase) GetAllUsers(ctx context.Context) ([]*domain.UserProfile, error) {
	users, err := uc.userRepo.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	profiles := make([]*domain.UserProfile, len(users))
	for i, user := range users {
		profiles[i] = &domain.UserProfile{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			Role:        user.Role,
			IsActive:    user.IsActive,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			LastLoginAt: user.LastLoginAt,
		}
	}

	return profiles, nil
}

// ActivateUser activates a user account
func (uc *UserUsecase) ActivateUser(ctx context.Context, userID string) error {
	return uc.userRepo.ActivateUser(ctx, userID)
}

// DeactivateUser deactivates a user account
func (uc *UserUsecase) DeactivateUser(ctx context.Context, userID string) error {
	return uc.userRepo.DeactivateUser(ctx, userID)
}

// ResetUserPassword resets a user's password (admin only)
func (uc *UserUsecase) ResetUserPassword(ctx context.Context, adminUserID, targetUserID, newPassword string) error {
	// Verify admin user has admin role
	adminUser, err := uc.userRepo.GetUserByID(ctx, adminUserID)
	if err != nil {
		return fmt.Errorf("admin user not found")
	}

	if adminUser.Role != "admin" {
		return fmt.Errorf("insufficient permissions")
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	err = uc.userRepo.UpdateUserPassword(ctx, targetUserID, string(newPasswordHash))
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return nil
}

// EnsureDefaultUser ensures the default admin user exists
func (uc *UserUsecase) EnsureDefaultUser(ctx context.Context) error {
	// Check if any admin user exists
	users, err := uc.userRepo.GetAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing users: %w", err)
	}

	// Check if we have any admin users
	hasAdmin := false
	for _, user := range users {
		if user.Role == "admin" && user.IsActive {
			hasAdmin = true
			break
		}
	}

	// If no admin exists, create default admin
	if !hasAdmin {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash default password: %w", err)
		}

		defaultAdmin := &domain.User{
			ID:        uuid.New().String(),
			Username:  "admin",
			Email:     "admin@loadtester.com",
			Password:  string(passwordHash),
			FirstName: "Default",
			LastName:  "Admin",
			Role:      "admin",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = uc.userRepo.CreateUser(ctx, defaultAdmin)
		if err != nil {
			return fmt.Errorf("failed to create default admin user: %w", err)
		}
	}

	return nil
}

// generateJWT generates a JWT token for a user
func (uc *UserUsecase) generateJWT(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(), // 24 hours
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwtSecret))
}

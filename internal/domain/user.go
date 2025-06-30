package domain

import (
	"context"
)

// UserUsecase defines the interface for user business logic
type UserUsecase interface {
	// Authentication
	Login(ctx context.Context, username, password string) (*User, string, error) // returns user, token, error
	AuthenticateUser(ctx context.Context, username, password string) (*AuthResponse, error)
	ValidateToken(ctx context.Context, token string) (*User, error)
	ValidateJWTToken(ctx context.Context, token string) (*UserProfile, error)

	// User management
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	GetUserProfile(ctx context.Context, userID string) (*UserProfile, error)
	UpdateUserProfile(ctx context.Context, userID string, req *UpdateUserRequest) (*UserProfile, error)
	ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error
	GetAllUsers(ctx context.Context) ([]*UserProfile, error)
	ActivateUser(ctx context.Context, userID string) error
	DeactivateUser(ctx context.Context, userID string) error

	// Admin functions
	ResetUserPassword(ctx context.Context, adminUserID, targetUserID, newPassword string) error

	// CLI functions
	EnsureDefaultUser(ctx context.Context) error
}

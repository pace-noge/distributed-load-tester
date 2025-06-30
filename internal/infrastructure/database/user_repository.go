package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
)

// UserRepository implements domain.UserRepository
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user in the database
func (r *UserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Email, user.Password,
		user.FirstName, user.LastName, user.Role, user.IsActive, user.CreatedAt, user.UpdatedAt)

	return err
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, role, is_active,
		       created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FirstName, &user.LastName, &user.Role, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	return user, err
}

// GetUserByUsername retrieves a user by username
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, role, is_active,
		       created_at, updated_at, last_login_at
		FROM users WHERE username = $1
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FirstName, &user.LastName, &user.Role, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	return user, err
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, role, is_active,
		       created_at, updated_at, last_login_at
		FROM users WHERE email = $1
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FirstName, &user.LastName, &user.Role, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	return user, err
}

// UpdateUser updates user information
func (r *UserRepository) UpdateUser(ctx context.Context, id string, updates *domain.UpdateUserRequest) (*domain.User, error) {
	setParts := []string{}
	args := []interface{}{}
	argCount := 1

	if updates.Email != "" {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argCount))
		args = append(args, updates.Email)
		argCount++
	}

	if updates.FirstName != "" {
		setParts = append(setParts, fmt.Sprintf("first_name = $%d", argCount))
		args = append(args, updates.FirstName)
		argCount++
	}

	if updates.LastName != "" {
		setParts = append(setParts, fmt.Sprintf("last_name = $%d", argCount))
		args = append(args, updates.LastName)
		argCount++
	}

	if updates.Role != "" {
		setParts = append(setParts, fmt.Sprintf("role = $%d", argCount))
		args = append(args, updates.Role)
		argCount++
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())
	argCount++

	// Add user ID for WHERE clause
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE id = $%d
	`, strings.Join(setParts, ", "), argCount)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("user not found")
	}

	// Return the updated user
	return r.GetUserByID(ctx, id)
}

// UpdateUserPassword updates user password
func (r *UserRepository) UpdateUserPassword(ctx context.Context, id string, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, passwordHash, time.Now(), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	query := `
		UPDATE users
		SET last_login_at = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// DeleteUser deletes a user
func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ListUsers retrieves users with pagination
func (r *UserRepository) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM users`
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get users
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, role, is_active,
		       created_at, updated_at, last_login_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.Password,
			&user.FirstName, &user.LastName, &user.Role, &user.IsActive,
			&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, rows.Err()
}

// EnsureDefaultUser creates the default admin user if it doesn't exist
func (r *UserRepository) EnsureDefaultUser(ctx context.Context) error {
	// Check if default user exists
	_, err := r.GetUserByUsername(ctx, "admin")
	if err == nil {
		// User already exists
		return nil
	}

	// Create default user
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash default password: %w", err)
	}

	defaultUser := &domain.User{
		ID:        uuid.New().String(),
		Username:  "admin",
		Email:     "admin@loadtester.local",
		Password:  string(passwordHash),
		FirstName: "System",
		LastName:  "Administrator",
		Role:      "admin",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return r.CreateUser(ctx, defaultUser)
}

// ResetDefaultUserPassword resets the default admin user password
func (r *UserRepository) ResetDefaultUserPassword(ctx context.Context, newPasswordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE username = 'admin'
	`

	result, err := r.db.ExecContext(ctx, query, newPasswordHash, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("default admin user not found")
	}

	return nil
}

// GetAllUsers retrieves all users from the database
func (r *UserRepository) GetAllUsers(ctx context.Context) ([]*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, role, is_active,
		       created_at, updated_at, last_login_at
		FROM users ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.Password,
			&user.FirstName, &user.LastName, &user.Role, &user.IsActive,
			&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// ActivateUser activates a user account
func (r *UserRepository) ActivateUser(ctx context.Context, userID string) error {
	query := `UPDATE users SET is_active = true, updated_at = $1 WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// DeactivateUser deactivates a user account
func (r *UserRepository) DeactivateUser(ctx context.Context, userID string) error {
	query := `UPDATE users SET is_active = false, updated_at = $1 WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

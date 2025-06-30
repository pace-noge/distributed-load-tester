package cmd

import (
	"context"
	"fmt"
	"log"
	"syscall"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/database"
	"github.com/pace-noge/distributed-load-tester/internal/user/usecase"
)

// NewUserCommand creates the user management CLI command
func NewUserCommand() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "User management commands",
		Subcommands: []*cli.Command{
			{
				Name:  "reset-password",
				Usage: "Reset the default admin user password",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "database-url",
						Usage:    "Database connection URL",
						EnvVars:  []string{"DATABASE_URL"},
						Required: true,
					},
					&cli.StringFlag{
						Name:  "password",
						Usage: "New password (if not provided, will prompt)",
					},
				},
				Action: resetDefaultPassword,
			},
			{
				Name:  "create-admin",
				Usage: "Ensure default admin user exists",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "database-url",
						Usage:    "Database connection URL",
						EnvVars:  []string{"DATABASE_URL"},
						Required: true,
					},
				},
				Action: createDefaultAdmin,
			},
		},
	}
}

// resetDefaultPassword resets the default admin user password
func resetDefaultPassword(c *cli.Context) error {
	databaseURL := c.String("database-url")
	password := c.String("password")

	// If password not provided, prompt for it
	if password == "" {
		fmt.Print("Enter new password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password = string(passwordBytes)
		fmt.Println() // Print newline after password input

		if len(password) < 8 {
			return fmt.Errorf("password must be at least 8 characters long")
		}
	}

	// Initialize database connection
	db, err := database.NewPostgresDB(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize repository and usecase
	userRepo := database.NewUserRepository(db.GetDB())
	userUsecase := usecase.NewUserUsecase(userRepo, "your-jwt-secret-key") // In production, use proper secret

	// Reset password
	ctx := context.Background()

	// First ensure default user exists
	err = userUsecase.EnsureDefaultUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to ensure default user exists: %w", err)
	}

	// Get default admin user
	users, err := userRepo.GetAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	var defaultAdminID string
	for _, user := range users {
		if user.Username == "admin" && user.Role == "admin" {
			defaultAdminID = user.ID
			break
		}
	}

	if defaultAdminID == "" {
		return fmt.Errorf("default admin user not found")
	}

	// Create password change request
	req := &domain.ChangePasswordRequest{
		CurrentPassword: "admin123", // Default password
		NewPassword:     password,
	}

	// Change password
	err = userUsecase.ChangePassword(ctx, defaultAdminID, req)
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	log.Println("Default admin password reset successfully")
	return nil
}

// createDefaultAdmin ensures the default admin user exists
func createDefaultAdmin(c *cli.Context) error {
	databaseURL := c.String("database-url")

	// Initialize database connection
	db, err := database.NewPostgresDB(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize repository and usecase
	userRepo := database.NewUserRepository(db.GetDB())
	userUsecase := usecase.NewUserUsecase(userRepo, "your-jwt-secret-key") // In production, use proper secret

	// Ensure default user exists
	ctx := context.Background()
	err = userUsecase.EnsureDefaultUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to ensure default user exists: %w", err)
	}

	log.Println("Default admin user ensured successfully")
	log.Println("Default credentials: username=admin, password=admin123")
	log.Println("Please change the default password using the 'user reset-password' command")
	return nil
}

package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	userUsecase "github.com/pace-noge/distributed-load-tester/internal/user/usecase"
)

// UserHandler handles HTTP requests for user management
type UserHandler struct {
	userUsecase *userUsecase.UserUsecase
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userUsecase *userUsecase.UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: userUsecase,
	}
}

// RegisterRoutes registers user management routes
func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	// Authentication routes
	mux.HandleFunc("/api/auth/login", h.handleCORS(h.handleLogin))
	mux.HandleFunc("/api/auth/profile", h.handleCORS(h.requireAuth(h.handleGetProfile)))
	mux.HandleFunc("/api/auth/change-password", h.handleCORS(h.requireAuth(h.handleChangePassword)))

	// User management routes (admin only)
	mux.HandleFunc("/api/users", h.handleCORS(h.requireAuth(h.requireAdmin(h.handleUsers))))
	mux.HandleFunc("/api/users/create", h.handleCORS(h.requireAuth(h.requireAdmin(h.handleCreateUser))))
	mux.HandleFunc("/api/users/", h.handleCORS(h.requireAuth(h.requireAdmin(h.handleUserByID))))
}

// handleLogin handles user authentication
func (h *UserHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	authResponse, err := h.userUsecase.AuthenticateUser(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse)
}

// handleGetProfile handles profile requests
func (h *UserHandler) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserProfileFromContext(r)
	if user == nil {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Return user without password hash
	response := map[string]interface{}{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"firstName":   user.FirstName,
		"lastName":    user.LastName,
		"role":        user.Role,
		"isActive":    user.IsActive,
		"createdAt":   user.CreatedAt,
		"updatedAt":   user.UpdatedAt,
		"lastLoginAt": user.LastLoginAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChangePassword handles password change requests
func (h *UserHandler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserProfileFromContext(r)
	if user == nil {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	var req domain.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := h.userUsecase.ChangePassword(r.Context(), user.ID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password changed successfully"})
}

// handleUsers handles user listing and profile updates
func (h *UserHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetAllUsers(w, r)
	case http.MethodPut:
		h.handleUpdateProfile(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetAllUsers handles getting all users
func (h *UserHandler) handleGetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userUsecase.GetAllUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return users without password hashes
	var response []map[string]interface{}
	for _, user := range users {
		response = append(response, map[string]interface{}{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"firstName":   user.FirstName,
			"lastName":    user.LastName,
			"role":        user.Role,
			"isActive":    user.IsActive,
			"createdAt":   user.CreatedAt,
			"updatedAt":   user.UpdatedAt,
			"lastLoginAt": user.LastLoginAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUpdateProfile handles profile update requests
func (h *UserHandler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := getUserProfileFromContext(r)
	if user == nil {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	var req domain.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	updatedUser, err := h.userUsecase.UpdateUserProfile(r.Context(), user.ID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return updated user without password hash
	response := map[string]interface{}{
		"id":        updatedUser.ID,
		"username":  updatedUser.Username,
		"email":     updatedUser.Email,
		"firstName": updatedUser.FirstName,
		"lastName":  updatedUser.LastName,
		"role":      updatedUser.Role,
		"isActive":  updatedUser.IsActive,
		"updatedAt": updatedUser.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateUser handles user creation
func (h *UserHandler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	user, err := h.userUsecase.CreateUser(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return created user without password hash
	response := map[string]interface{}{
		"id":        user.ID,
		"username":  user.Username,
		"email":     user.Email,
		"firstName": user.FirstName,
		"lastName":  user.LastName,
		"role":      user.Role,
		"isActive":  user.IsActive,
		"createdAt": user.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleUserByID handles operations on specific users by ID
func (h *UserHandler) handleUserByID(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userID := pathParts[3]

	switch r.Method {
	case http.MethodGet:
		h.handleGetUserByID(w, r, userID)
	case http.MethodPost:
		// Handle activate/deactivate
		if strings.HasSuffix(r.URL.Path, "/activate") {
			h.handleActivateUser(w, r, userID)
		} else if strings.HasSuffix(r.URL.Path, "/deactivate") {
			h.handleDeactivateUser(w, r, userID)
		} else {
			http.Error(w, "Invalid operation", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetUserByID handles getting a specific user by ID
func (h *UserHandler) handleGetUserByID(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := h.userUsecase.GetUserProfile(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return user without password hash
	response := map[string]interface{}{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"firstName":   user.FirstName,
		"lastName":    user.LastName,
		"role":        user.Role,
		"isActive":    user.IsActive,
		"createdAt":   user.CreatedAt,
		"updatedAt":   user.UpdatedAt,
		"lastLoginAt": user.LastLoginAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleActivateUser handles user activation
func (h *UserHandler) handleActivateUser(w http.ResponseWriter, r *http.Request, userID string) {
	err := h.userUsecase.ActivateUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User activated successfully"})
}

// handleDeactivateUser handles user deactivation
func (h *UserHandler) handleDeactivateUser(w http.ResponseWriter, r *http.Request, userID string) {
	err := h.userUsecase.DeactivateUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User deactivated successfully"})
}

// requireAuth middleware checks if user is authenticated
func (h *UserHandler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Bearer token required", http.StatusUnauthorized)
			return
		}

		user, err := h.userUsecase.ValidateJWTToken(r.Context(), tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := setUserProfileInContext(r.Context(), user)
		next(w, r.WithContext(ctx))
	}
}

// requireAdmin middleware checks if user has admin role
func (h *UserHandler) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUserProfileFromContext(r)
		if user == nil || user.Role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// handleCORS handles CORS headers
func (h *UserHandler) handleCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

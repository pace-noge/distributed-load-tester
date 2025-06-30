package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/auth"

	"github.com/gorilla/mux"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	masterUsecase "github.com/pace-noge/distributed-load-tester/internal/master/usecase"
	pb "github.com/pace-noge/distributed-load-tester/proto" // Import generated protobuf
)

// HTTPHandler handles HTTP requests for the Master service.
type HTTPHandler struct {
	Router    *mux.Router
	usecase   *masterUsecase.MasterUsecase
	jwtSecret string
}

// NewHTTPHandler creates a new HTTPHandler instance.
func NewHTTPHandler(uc *masterUsecase.MasterUsecase, jwtSecret string) *HTTPHandler {
	h := &HTTPHandler{
		usecase:   uc,
		jwtSecret: jwtSecret,
	}
	r := mux.NewRouter()

	// CORS middleware
	r.Use(h.corsMiddleware)

	// API routes (protected by auth middleware)
	api := r.PathPrefix("/api").Subrouter()
	api.Use(h.authMiddleware)
	api.HandleFunc("/test/submit", h.submitTest).Methods("POST")
	api.HandleFunc("/dashboard", h.getDashboardStatus).Methods("GET")
	api.HandleFunc("/tests", h.getTests).Methods("GET")
	api.HandleFunc("/tests/{testId}/results", h.getTestResults).Methods("GET")
	api.HandleFunc("/tests/{testId}/aggregated-result", h.getAggregatedTestResult).Methods("GET")
	api.HandleFunc("/tests/{testId}/aggregate", h.triggerAggregation).Methods("POST")

	// Authentication route (public)
	r.HandleFunc("/auth/login", h.login).Methods("POST")

	h.Router = r
	return h
}

// RegisterWebSocketHandler registers the WebSocket handler before the static file handler
func (h *HTTPHandler) RegisterWebSocketHandler(wsHandler func(http.ResponseWriter, *http.Request)) {
	// Register WebSocket route before adding static file handler
	h.Router.HandleFunc("/ws", wsHandler).Methods("GET")

	// Now add the static file handler as the catch-all
	// This must be done after all specific routes are registered
	h.Router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/dist")))
}

// corsMiddleware handles CORS headers.
func (h *HTTPHandler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust in production
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authMiddleware validates JWT tokens.
func (h *HTTPHandler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader { // No "Bearer " prefix found
			http.Error(w, "Invalid token format: Bearer token required", http.StatusUnauthorized)
			return
		}

		userID, err := auth.ValidateJWT(tokenString) // Use the shared auth package
		if err != nil {
			log.Printf("JWT validation failed: %v", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add userID to context for downstream handlers
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// login handles user login and issues a JWT token.
func (h *HTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// In a real application, you'd validate username/password against a database.
	// For this example, we'll use a very simple hardcoded check.
	if credentials.Username == "admin" && credentials.Password == "password" {
		token, err := auth.GenerateJWT("admin") // Use the shared auth package
		if err != nil {
			log.Printf("Failed to generate JWT: %v", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"token": token, "message": "Login successful"})
		return
	}

	http.Error(w, "Invalid username or password", http.StatusUnauthorized)
}

// submitTest handles requests to submit a new load test.
func (h *HTTPHandler) submitTest(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized: User ID not found in context", http.StatusUnauthorized)
		return
	}

	var req pb.TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	req.RequesterId = userID // Set requester ID from authenticated user

	// Call the gRPC method directly via the usecase
	resp, err := h.usecase.SubmitTest(r.Context(), &domain.TestRequest{
		Name:              req.Name,
		VegetaPayloadJSON: req.VegetaPayloadJson,
		DurationSeconds:   req.DurationSeconds,
		RatePerSecond:     req.RatePerSecond,
		TargetsBase64:     req.TargetsBase64,
		RequesterID:       req.RequesterId,
		WorkerCount:       req.WorkerCount,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit test: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"testId": resp, "message": "Test submitted successfully"})
}

// getDashboardStatus provides dashboard data.
func (h *HTTPHandler) getDashboardStatus(w http.ResponseWriter, r *http.Request) {
	dashboard, err := h.usecase.GetDashboardStatus(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get dashboard status: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(dashboard)
}

// getTests retrieves a list of tests with optional pagination.
func (h *HTTPHandler) getTests(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	
	// Default values
	page := 1
	limit := 20
	
	// Parse page parameter
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	// Parse limit parameter
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	// Calculate offset
	offset := (page - 1) * limit
	
	// Get paginated results
	tests, totalCount, err := h.usecase.GetTestRequestsPaginated(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tests: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Calculate pagination metadata
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	
	// Create response with pagination metadata
	response := map[string]interface{}{
		"tests":       tests,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
			"has_prev":    page > 1,
			"has_next":    page < totalPages,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getTestResults retrieves raw results for a specific test.
func (h *HTTPHandler) getTestResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["testId"]
	if testID == "" {
		http.Error(w, "Test ID is required", http.StatusBadRequest)
		return
	}

	results, err := h.usecase.GetRawTestResults(r.Context(), testID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get test results: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
}

// getAggregatedTestResult retrieves the aggregated result for a specific test.
func (h *HTTPHandler) getAggregatedTestResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["testId"]
	if testID == "" {
		http.Error(w, "Test ID is required", http.StatusBadRequest)
		return
	}

	aggregatedResult, err := h.usecase.GetAggregatedTestResult(r.Context(), testID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf("Aggregated result not found for test %s. Results may still be processing.", testID), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get aggregated test result: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(aggregatedResult)
}

// triggerAggregation manually triggers aggregation for a specific test.
func (h *HTTPHandler) triggerAggregation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["testId"]
	if testID == "" {
		http.Error(w, "Test ID is required", http.StatusBadRequest)
		return
	}

	// Trigger aggregation in a goroutine
	go h.usecase.TriggerAggregation(context.Background(), testID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Aggregation triggered for test %s", testID),
	})
}

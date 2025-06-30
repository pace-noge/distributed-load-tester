package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
	masterUsecase "github.com/pace-noge/distributed-load-tester/internal/master/usecase"
	userHttp "github.com/pace-noge/distributed-load-tester/internal/user/delivery/http"
	userUsecase "github.com/pace-noge/distributed-load-tester/internal/user/usecase"
	pb "github.com/pace-noge/distributed-load-tester/proto" // Import generated protobuf
)

// Define context key type at package level to avoid conflicts
type contextKey string

const userContextKey contextKey = "user"

// HTTPHandler handles HTTP requests for the Master service.
type HTTPHandler struct {
	Router      *mux.Router
	usecase     *masterUsecase.MasterUsecase
	userUsecase *userUsecase.UserUsecase
	jwtSecret   string
}

// NewHTTPHandler creates a new HTTPHandler instance.
func NewHTTPHandler(uc *masterUsecase.MasterUsecase, userUc *userUsecase.UserUsecase, jwtSecret string) *HTTPHandler {
	h := &HTTPHandler{
		usecase:     uc,
		userUsecase: userUc,
		jwtSecret:   jwtSecret,
	}
	r := mux.NewRouter()

	// CORS middleware
	r.Use(h.corsMiddleware)

	// Register user management routes with their own prefix
	userHandler := userHttp.NewUserHandler(userUc)
	userMux := http.NewServeMux()
	userHandler.RegisterRoutes(userMux)

	// Mount user routes specifically at /api prefix
	r.PathPrefix("/api/auth").Handler(userMux)
	r.PathPrefix("/api/users").Handler(userMux)

	// API routes (protected by auth middleware)
	api := r.PathPrefix("/api").Subrouter()
	api.Use(h.authMiddleware)
	api.HandleFunc("/test/submit", h.submitTest).Methods("POST")
	api.HandleFunc("/dashboard", h.getDashboardStatus).Methods("GET")
	api.HandleFunc("/tests", h.getTests).Methods("GET")
	api.HandleFunc("/tests/{testId}/results", h.getTestResults).Methods("GET")
	api.HandleFunc("/tests/{testId}/aggregated-result", h.getAggregatedTestResult).Methods("GET")
	api.HandleFunc("/tests/{testId}/aggregate", h.triggerAggregation).Methods("POST")

	// Sharing and inbox endpoints
	api.HandleFunc("/tests/{testId}/share", h.shareTest).Methods("POST")
	api.HandleFunc("/shared/{linkId}", h.accessSharedLink).Methods("GET")
	api.HandleFunc("/inbox", h.getInbox).Methods("GET")
	api.HandleFunc("/inbox/{linkId}/read", h.markInboxItemRead).Methods("POST")

	// Analytics routes
	api.HandleFunc("/analytics/overview", h.getAnalyticsOverview).Methods("GET")
	api.HandleFunc("/analytics/targets", h.getTargetAnalytics).Methods("GET")

	h.Router = r
	return h
}

// RegisterWebSocketHandler registers the WebSocket handler before the static file handler
func (h *HTTPHandler) RegisterWebSocketHandler(wsHandler func(http.ResponseWriter, *http.Request)) {
	// Register WebSocket route before adding static file handler
	h.Router.HandleFunc("/ws", wsHandler).Methods("GET")

	// Serve static files for the frontend (this should be last)
	// Use a custom handler to serve index.html for client-side routing
	staticHandler := http.FileServer(http.Dir("./frontend/dist"))
	h.Router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For client-side routing, serve index.html for non-API routes
		if !strings.HasPrefix(r.URL.Path, "/api") && !strings.HasPrefix(r.URL.Path, "/ws") && !strings.Contains(r.URL.Path, ".") {
			http.ServeFile(w, r, "./frontend/dist/index.html")
			return
		}
		staticHandler.ServeHTTP(w, r)
	})
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

		// Use the user management system for token validation
		user, err := h.userUsecase.ValidateJWTToken(r.Context(), tokenString)
		if err != nil {
			log.Printf("JWT validation failed: %v", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add user to context for downstream handlers
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// submitTest handles requests to submit a new load test.
func (h *HTTPHandler) submitTest(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		http.Error(w, "Unauthorized: User not found in context", http.StatusUnauthorized)
		return
	}

	var req pb.TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	req.RequesterId = user.ID // Set requester ID from authenticated user

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
	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		http.Error(w, "Unauthorized: User not found in context", http.StatusUnauthorized)
		return
	}

	// Parse pagination params
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	tests, total, err := h.usecase.GetTestRequestsPaginatedByUser(r.Context(), user.ID, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tests: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"tests":  tests,
		"total":  total,
		"limit":  limit,
		"offset": offset,
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

// getAnalyticsOverview provides comprehensive analytics overview
func (h *HTTPHandler) getAnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for time range
	query := r.URL.Query()

	var req domain.AnalyticsRequest

	// Parse optional time range
	startDateStr := query.Get("startDate")
	endDateStr := query.Get("endDate")

	if startDateStr != "" && endDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			http.Error(w, "Invalid start date format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end date format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		req.TimeRange = &domain.AnalyticsTimeRange{
			StartDate: startDate,
			EndDate:   endDate,
		}
	}

	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		http.Error(w, "Unauthorized: User not found in context", http.StatusUnauthorized)
		return
	}

	req.UserID = user.ID

	overview, err := h.usecase.GetAnalyticsOverview(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get analytics overview: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

// getTargetAnalytics provides analytics for specific targets
func (h *HTTPHandler) getTargetAnalytics(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	var req domain.AnalyticsRequest

	// Parse optional target URL filter
	req.TargetURL = query.Get("target")

	// Parse optional time range
	startDateStr := query.Get("startDate")
	endDateStr := query.Get("endDate")

	if startDateStr != "" && endDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			http.Error(w, "Invalid start date format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end date format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		req.TimeRange = &domain.AnalyticsTimeRange{
			StartDate: startDate,
			EndDate:   endDate,
		}
	}

	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		http.Error(w, "Unauthorized: User not found in context", http.StatusUnauthorized)
		return
	}

	req.UserID = user.ID

	targetAnalytics, err := h.usecase.GetTargetAnalytics(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get target analytics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targetAnalytics)
}

// shareTest handles sharing a test and returns a shareable link.
func (h *HTTPHandler) shareTest(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		http.Error(w, "Unauthorized: User not found in context", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	testID := vars["testId"]
	if testID == "" {
		http.Error(w, "Test ID is required", http.StatusBadRequest)
		return
	}
	// Check for optional userId query param
	userIdParam := r.URL.Query().Get("userId")
	var link *domain.SharedLink
	var err error
	if userIdParam != "" {
		// Share to another user's inbox
		link, err = h.usecase.ShareTestToUserInbox(r.Context(), testID, user.ID, userIdParam)
	} else {
		// Regular share (generate link only)
		link, err = h.usecase.ShareTest(r.Context(), testID, user.ID)
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to share test: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"link": "/api/shared/" + link.ID, "expiresAt": link.ExpiresAt.Format(time.RFC3339)})
}

// accessSharedLink allows a user to access a shared test link and adds it to their history.
func (h *HTTPHandler) accessSharedLink(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		http.Error(w, "Unauthorized: User not found in context", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	linkID := vars["linkId"]
	if linkID == "" {
		http.Error(w, "Link ID is required", http.StatusBadRequest)
		return
	}
	test, err := h.usecase.AccessSharedLink(r.Context(), linkID, user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to access shared link: %v", err), http.StatusForbidden)
		return
	}
	json.NewEncoder(w).Encode(test)
}

// getInbox returns the user's inbox of shared tests.
func (h *HTTPHandler) getInbox(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		http.Error(w, "Unauthorized: User not found in context", http.StatusUnauthorized)
		return
	}
	inbox, err := h.usecase.GetInbox(r.Context(), user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get inbox: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"inbox": inbox})
}

// markInboxItemRead marks a shared inbox item as read for the user.
func (h *HTTPHandler) markInboxItemRead(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		http.Error(w, "Unauthorized: User not found in context", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	linkID := vars["linkId"]
	if linkID == "" {
		http.Error(w, "Link ID is required", http.StatusBadRequest)
		return
	}
	if err := h.usecase.MarkInboxItemRead(r.Context(), linkID, user.ID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to mark inbox item as read: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pace-noge/distributed-load-tester/internal/infrastructure/auth"
	masterUsecase "github.com/pace-noge/distributed-load-tester/internal/master/usecase"
)

// WebSocketHandler handles WebSocket connections for real-time dashboard updates
type WebSocketHandler struct {
	masterUsecase *masterUsecase.MasterUsecase
	jwtSecretKey  string
	upgrader      websocket.Upgrader
	clients       map[*websocket.Conn]bool
	broadcast     chan []byte
	register      chan *websocket.Conn
	unregister    chan *websocket.Conn
	mu            sync.RWMutex
}

// DashboardMessage represents the message structure sent via WebSocket
type DashboardMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(masterUC *masterUsecase.MasterUsecase, jwtSecretKey string) *WebSocketHandler {
	return &WebSocketHandler{
		masterUsecase: masterUC,
		jwtSecretKey:  jwtSecretKey,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, you should check the origin properly
				return true
			},
		},
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// StartHub starts the WebSocket hub that manages client connections and broadcasts
func (h *WebSocketHandler) StartHub(ctx context.Context) {
	// Start the periodic dashboard update broadcaster
	go h.startDashboardBroadcaster(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("WebSocket hub shutting down")
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client registered. Total clients: %d", len(h.clients))

			// Send initial dashboard data to new client
			h.sendDashboardDataToClient(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Printf("WebSocket client unregistered. Total clients: %d", len(h.clients))

		case data := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case <-ctx.Done():
					h.mu.RUnlock()
					return
				default:
					err := client.WriteMessage(websocket.TextMessage, data)
					if err != nil {
						log.Printf("Error writing to WebSocket client: %v", err)
						client.Close()
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket connection upgrades and manages client connections
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authenticate the WebSocket connection
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	_, err := auth.ValidateJWT(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Register the new client
	h.register <- conn

	// Handle client disconnection and cleanup
	defer func() {
		h.unregister <- conn
	}()

	// Handle incoming messages from client and keep connection alive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send periodic pings to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Handle messages in a goroutine
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Error sending ping to WebSocket client: %v", err)
					return
				}
			}
		}
	}()

	// Read loop for incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// startDashboardBroadcaster periodically fetches dashboard data and broadcasts to all clients
func (h *WebSocketHandler) startDashboardBroadcaster(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second) // Update every 2 seconds for real-time feel
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dashboardData, err := h.masterUsecase.GetDashboardStatus(ctx)
			if err != nil {
				log.Printf("Error fetching dashboard data for broadcast: %v", err)
				continue
			}

			message := DashboardMessage{
				Type: "dashboard_update",
				Data: dashboardData,
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshaling dashboard data for broadcast: %v", err)
				continue
			}

			// Only broadcast if there are connected clients
			h.mu.RLock()
			if len(h.clients) > 0 {
				select {
				case h.broadcast <- data:
				case <-ctx.Done():
					h.mu.RUnlock()
					return
				default:
					// Broadcast channel is full, skip this update
					log.Println("Broadcast channel full, skipping dashboard update")
				}
			}
			h.mu.RUnlock()
		}
	}
}

// sendDashboardDataToClient sends initial dashboard data to a specific client
func (h *WebSocketHandler) sendDashboardDataToClient(client *websocket.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dashboardData, err := h.masterUsecase.GetDashboardStatus(ctx)
	if err != nil {
		log.Printf("Error fetching dashboard data for new client: %v", err)
		return
	}

	message := DashboardMessage{
		Type: "dashboard_update",
		Data: dashboardData,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling dashboard data for new client: %v", err)
		return
	}

	err = client.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Printf("Error sending initial dashboard data to new client: %v", err)
	}
}

// BroadcastTestUpdate broadcasts test-related updates to all connected clients
func (h *WebSocketHandler) BroadcastTestUpdate(testUpdate interface{}) {
	message := DashboardMessage{
		Type: "test_update",
		Data: testUpdate,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling test update for broadcast: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Println("Broadcast channel full, skipping test update")
	}
}

package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

// PaymentEvent represents a payment status update event
type PaymentEvent struct {
	Type      string    `json:"type"`      // payment.pending, payment.confirming, payment.completed, payment.expired, payment.failed
	PaymentID string    `json:"payment_id"`
	Status    string    `json:"status"`
	TxHash    string    `json:"tx_hash,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// PaymentWSHandler handles WebSocket connections for payment status updates
type PaymentWSHandler struct {
	redisClient *redis.Client
	upgrader    websocket.Upgrader

	// Connection management
	mu          sync.RWMutex
	connections map[string]map[*websocket.Conn]bool // paymentID -> connections
}

// NewPaymentWSHandler creates a new WebSocket handler for payment status
func NewPaymentWSHandler(redisClient *redis.Client) *PaymentWSHandler {
	return &PaymentWSHandler{
		redisClient: redisClient,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow all origins for now (should be restricted in production)
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connections: make(map[string]map[*websocket.Conn]bool),
	}
}

// HandlePaymentStatus handles WebSocket connections for a specific payment
// GET /ws/payments/:id
func (h *PaymentWSHandler) HandlePaymentStatus(c *gin.Context) {
	ctx := c.Request.Context()
	paymentID := c.Param("id")

	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Payment ID is required",
			},
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		}).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Register connection
	h.registerConnection(paymentID, conn)
	defer h.unregisterConnection(paymentID, conn)

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"payment_id":  paymentID,
		"remote_addr": c.Request.RemoteAddr,
	}).Info("WebSocket connection established")

	// Send initial connection confirmation
	initEvent := PaymentEvent{
		Type:      "connection.established",
		PaymentID: paymentID,
		Timestamp: time.Now(),
		Message:   "Connected to payment status updates",
	}
	if err := conn.WriteJSON(initEvent); err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		}).Error("Failed to send initial event")
		return
	}

	// Subscribe to Redis pub/sub for this payment
	channelName := fmt.Sprintf("payment_events:%s", paymentID)
	pubsub := h.redisClient.Subscribe(ctx, channelName)
	defer pubsub.Close()

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"payment_id": paymentID,
		"channel":    channelName,
	}).Debug("Subscribed to Redis channel")

	// Handle WebSocket communication
	h.handleConnection(ctx, conn, pubsub, paymentID)
}

// handleConnection manages the WebSocket connection lifecycle
func (h *PaymentWSHandler) handleConnection(
	ctx context.Context,
	conn *websocket.Conn,
	pubsub *redis.PubSub,
	paymentID string,
) {
	// Create channels for graceful shutdown
	done := make(chan struct{})
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Start goroutine to read from WebSocket (for pong messages and close frames)
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.WithContext(ctx).WithFields(logrus.Fields{
						"error":      err.Error(),
						"payment_id": paymentID,
					}).Warn("WebSocket unexpected close")
				}
				return
			}
		}
	}()

	// Main event loop
	for {
		select {
		case <-done:
			// Connection closed by client
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"payment_id": paymentID,
			}).Debug("WebSocket connection closed by client")
			return

		case <-ctx.Done():
			// Context cancelled
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"payment_id": paymentID,
			}).Debug("WebSocket context cancelled")
			return

		case msg := <-pubsub.Channel():
			// Received event from Redis
			var event PaymentEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"error":      err.Error(),
					"payment_id": paymentID,
					"payload":    msg.Payload,
				}).Error("Failed to unmarshal payment event")
				continue
			}

			// Send event to WebSocket client
			if err := conn.WriteJSON(event); err != nil {
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"error":      err.Error(),
					"payment_id": paymentID,
				}).Error("Failed to send event to WebSocket client")
				return
			}

			logger.WithContext(ctx).WithFields(logrus.Fields{
				"payment_id": paymentID,
				"event_type": event.Type,
				"status":     event.Status,
			}).Debug("Sent payment event to WebSocket client")

		case <-pingTicker.C:
			// Send ping to keep connection alive
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"error":      err.Error(),
					"payment_id": paymentID,
				}).Error("Failed to send ping")
				return
			}
		}
	}
}

// registerConnection adds a connection to the tracking map
func (h *PaymentWSHandler) registerConnection(paymentID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[paymentID] == nil {
		h.connections[paymentID] = make(map[*websocket.Conn]bool)
	}
	h.connections[paymentID][conn] = true
}

// unregisterConnection removes a connection from the tracking map
func (h *PaymentWSHandler) unregisterConnection(paymentID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if connections, exists := h.connections[paymentID]; exists {
		delete(connections, conn)
		if len(connections) == 0 {
			delete(h.connections, paymentID)
		}
	}

	// Close the connection
	conn.Close()

	logger.Info("WebSocket connection unregistered", logger.Fields{
		"payment_id": paymentID,
	})
}

// GetActiveConnections returns the number of active connections for a payment
func (h *PaymentWSHandler) GetActiveConnections(paymentID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if connections, exists := h.connections[paymentID]; exists {
		return len(connections)
	}
	return 0
}

// GetTotalConnections returns the total number of active WebSocket connections
func (h *PaymentWSHandler) GetTotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, connections := range h.connections {
		total += len(connections)
	}
	return total
}

// PublishPaymentEvent publishes a payment event to Redis for broadcasting to WebSocket clients
// This function is used by the PaymentService to publish events
func PublishPaymentEvent(ctx context.Context, redisClient *redis.Client, event PaymentEvent) error {
	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal payment event: %w", err)
	}

	// Publish to Redis channel
	channelName := fmt.Sprintf("payment_events:%s", event.PaymentID)
	if err := redisClient.Publish(ctx, channelName, eventJSON).Err(); err != nil {
		return fmt.Errorf("failed to publish payment event to Redis: %w", err)
	}

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"payment_id": event.PaymentID,
		"event_type": event.Type,
		"channel":    channelName,
	}).Debug("Published payment event to Redis")

	return nil
}

package websocket

import (
	"notificationservice/config"
	"notificationservice/internal/kafka"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type WebSocketManager struct {
	clients   map[string]*websocket.Conn // Map user IDs to WebSocket connections
	broadcast chan Message
	mutex     *sync.Mutex
	logger    *zap.Logger
	producer  *kafka.Producer
	workers   int
}

type Message struct {
	Sender     string   `json:"sender"`
	Recipients []string `json:"recipients"`
	Message    string   `json:"message"`
}

func NewWebSocketManager(logger *zap.Logger, producer *kafka.Producer, workers int) *WebSocketManager {
	return &WebSocketManager{
		clients:   make(map[string]*websocket.Conn),
		broadcast: make(chan Message),
		mutex:     &sync.Mutex{},
		logger:    logger,
		producer:  producer,
		workers:   workers,
	}
}

func (m *WebSocketManager) HandleConnections(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		m.logger.Info("WebSocket upgrade requested", zap.String("path", c.Path()))
		return c.Next()
	}
	m.logger.Error("WebSocket upgrade required", zap.String("path", c.Path()))
	return fiber.ErrUpgradeRequired
}

func (m *WebSocketManager) WebSocket(c *websocket.Conn) {
	defer func() {
		m.mutex.Lock()
		for userID, conn := range m.clients {
			if conn == c {
				delete(m.clients, userID)
				break
			}
		}
		m.mutex.Unlock()
		m.logger.Info("Client disconnected", zap.String("remote_addr", c.RemoteAddr().String()))
		c.Close()
	}()

	// Validate JWT token
	tokenString := c.Query("token")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		m.logger.Error("Invalid token", zap.String("token", tokenString), zap.Error(err))
		c.WriteMessage(websocket.TextMessage, []byte("Invalid token"))
		c.Close()

		logEvent := map[string]interface{}{
			"message": "Invalid token",
			"token":   tokenString,
			"time":    time.Now().Format(time.RFC3339),
		}
		if logErr := m.producer.SendMessageToLogsTopic("Token", logEvent); logErr != nil {
			m.logger.Error("Failed to send log event", zap.Error(logErr))
		}
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["id"].(string)
	// Send welcome message via websocket
	m.logger.Info("Client connected", zap.String("remote_addr", c.RemoteAddr().String()), zap.String("user_id", userID))
	c.WriteMessage(websocket.TextMessage, []byte("Welcome back!"))

	logEvent := map[string]interface{}{
		"message":     "Client connected",
		"userId":      userID,
		"remote_addr": c.RemoteAddr().String(),
		"time":        time.Now().Format(time.RFC3339),
	}
	if logErr := m.producer.SendMessageToLogsTopic(userID, logEvent); logErr != nil {
		m.logger.Error("Failed to send log event", zap.Error(logErr))
	}
	m.mutex.Lock()
	if _, ok := m.clients[userID]; ok {
		m.CloseConnection(userID)
	}

	m.clients[userID] = c
	m.mutex.Unlock()

	for {
		var msg Message
		if err := c.ReadJSON(&msg); err != nil {
			m.logger.Error("Error reading JSON", zap.Error(err))
			break
		}
		m.logger.Info("Message received", zap.String("sender", msg.Sender), zap.Strings("recipients", msg.Recipients), zap.String("message", msg.Message))
		m.broadcast <- msg
	}
}

func (m *WebSocketManager) HandleMessages(msg Message) {
	m.logger.Info("Handling message", zap.String("sender", msg.Sender), zap.Strings("recipients", msg.Recipients), zap.String("message", msg.Message))
	m.mutex.Lock()
	for _, recipient := range msg.Recipients {
		if client, exists := m.clients[recipient]; exists {
			err := client.WriteJSON(msg.Message)
			if err != nil {
				m.logger.Error("Error writing JSON", zap.Error(err))
				client.Close()
				delete(m.clients, recipient)
			}
		} else {
			m.logger.Error("Recipient not connected", zap.String("recipient", recipient))
		}
	}
	m.mutex.Unlock()
}

func (m *WebSocketManager) StartWorkerPool() {
	for i := 0; i < m.workers; i++ {
		go m.worker(i)
	}
}

func (m *WebSocketManager) worker(id int) {
	for msg := range m.broadcast {
		m.logger.Info("Worker handling message", zap.Int("worker_id", id), zap.String("sender", msg.Sender), zap.Strings("recipients", msg.Recipients), zap.String("message", msg.Message))
		m.pushMessageToKafka(msg)
	}
}

func (m *WebSocketManager) pushMessageToKafka(msg Message) {
	msgEvent := Message{
		Sender:     msg.Sender,
		Recipients: msg.Recipients,
		Message:    msg.Message,
	}

	err := m.producer.SendMessageToMessageTopic(msg.Sender, msgEvent)
	if err != nil {
		m.logger.Error("Error writing message to Kafka", zap.Error(err))
	} else {
		m.logger.Info("Message pushed to Kafka", zap.String("sender", msg.Sender))
	}
}

func (m *WebSocketManager) CloseConnection(userId string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	client, ok := m.clients[userId]
	if !ok {
		m.logger.Error("User not connected", zap.String("userId", userId))
		return
	}

	if err := client.Close(); err != nil {
		m.logger.Error("Error closing WebSocket connection", zap.Error(err))
		logEvent := map[string]interface{}{
			"message": "Error closing WebSocket connection",
			"userId":  userId,
			"error":   err.Error(),
			"time":    time.Now().Format(time.RFC3339),
		}
		if logErr := m.producer.SendMessageToLogsTopic(userId, logEvent); logErr != nil {
			m.logger.Error("Failed to send log event", zap.Error(logErr))
		}
	}

	delete(m.clients, userId)
	m.logger.Info("WebSocket connection closed", zap.String("userId", userId))
	logEvent := map[string]interface{}{
		"message": "WebSocket connection closed",
		"userId":  userId,
		"time":    time.Now().Format(time.RFC3339),
	}
	if logErr := m.producer.SendMessageToLogsTopic(userId, logEvent); logErr != nil {
		m.logger.Error("Failed to send log event", zap.Error(logErr))
	}
}

package server

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ifrunruhin12/tasktime/internal/models"
)

type UserPresence struct {
	Username    string    `json:"username"`
	ConnectedAt time.Time `json:"connected_at"`
	LastPing    time.Time `json:"last_ping"`
}

type ConnectionManager struct {
	connections map[string]map[string]*websocket.Conn // username -> connectionID -> connection
	users       map[string]*UserPresence              // username -> presence info
	mu          sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]map[string]*websocket.Conn),
		users:       make(map[string]*UserPresence),
	}
}

func (cm *ConnectionManager) AddUser(username string, conn *websocket.Conn) string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Generate unique connection ID
	connectionID := generateConnectionID()

	// Initialize user connections map if it doesn't exist
	if cm.connections[username] == nil {
		cm.connections[username] = make(map[string]*websocket.Conn)
	}

	// Add the new connection
	cm.connections[username][connectionID] = conn

	// Update or create user presence (only if this is the first connection)
	if len(cm.connections[username]) == 1 {
		cm.users[username] = &UserPresence{
			Username:    username,
			ConnectedAt: time.Now(),
			LastPing:    time.Now(),
		}
	}

	log.Printf("User %s added connection %s. Total connections for user: %d, Total users: %d",
		username, connectionID, len(cm.connections[username]), len(cm.users))

	return connectionID
}

func (cm *ConnectionManager) RemoveUser(username, connectionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if userConnections, exists := cm.connections[username]; exists {
		if conn, connExists := userConnections[connectionID]; connExists {
			conn.Close()
			delete(userConnections, connectionID)

			// If no more connections for this user, remove from users map
			if len(userConnections) == 0 {
				delete(cm.connections, username)
				delete(cm.users, username)
				log.Printf("User %s completely disconnected. Total users: %d", username, len(cm.users))
			} else {
				log.Printf("User %s removed connection %s. Remaining connections: %d",
					username, connectionID, len(userConnections))
			}
		}
	}
}

func (cm *ConnectionManager) GetOnlineUsers() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	users := make([]string, 0, len(cm.users))
	for username := range cm.users {
		users = append(users, username)
	}
	return users
}

func (cm *ConnectionManager) GetOnlineUserCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.users)
}

func (cm *ConnectionManager) BroadcastToAll(message models.WSMessage) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var toRemove []struct {
		username     string
		connectionID string
	}

	for username, userConnections := range cm.connections {
		for connectionID, conn := range userConnections {
			err := conn.WriteJSON(message)
			if err != nil {
				log.Printf("Broadcast error: failed to send message to user %s connection %s: %v", username, connectionID, err)
				toRemove = append(toRemove, struct {
					username     string
					connectionID string
				}{username, connectionID})
			}
		}
	}

	if len(toRemove) > 0 {
		cm.mu.RUnlock()
		for _, item := range toRemove {
			cm.RemoveUser(item.username, item.connectionID)
		}
		cm.mu.RLock()
	}
}

func (cm *ConnectionManager) BroadcastToUser(username string, message models.WSMessage) error {
	cm.mu.RLock()
	userConnections, exists := cm.connections[username]
	cm.mu.RUnlock()

	if !exists {
		return nil // User not connected, silently ignore
	}

	var errors []error
	for connectionID, conn := range userConnections {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("Broadcast error: failed to send message to user %s connection %s: %v", username, connectionID, err)
			cm.RemoveUser(username, connectionID)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return errors[0] // Return first error
	}
	return nil
}

func (cm *ConnectionManager) UpdateLastPing(username string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if presence, exists := cm.users[username]; exists {
		presence.LastPing = time.Now()
	}
}

func (cm *ConnectionManager) GetConnection(username string) (*websocket.Conn, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if userConnections, exists := cm.connections[username]; exists && len(userConnections) > 0 {
		// Return any connection (first one found)
		for _, conn := range userConnections {
			return conn, true
		}
	}
	return nil, false
}

func (cm *ConnectionManager) IsUserOnline(username string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, exists := cm.users[username]
	return exists
}

// generateConnectionID creates a unique connection identifier
func generateConnectionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

package server

import (
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
	connections map[string]*websocket.Conn // username -> connection
	users       map[string]*UserPresence   // username -> presence info
	mu          sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*websocket.Conn),
		users:       make(map[string]*UserPresence),
	}
}

func (cm *ConnectionManager) AddUser(username string, conn *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if existingConn, exists := cm.connections[username]; exists {
		log.Printf("User %s already connected, closing old connection", username)
		existingConn.Close()
	}

	cm.connections[username] = conn
	cm.users[username] = &UserPresence{
		Username:    username,
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
	}

	log.Printf("User %s added to ConnectionManager. Total users: %d", username, len(cm.users))
}

func (cm *ConnectionManager) RemoveUser(username string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.connections[username]; exists {
		conn.Close()
		delete(cm.connections, username)
		delete(cm.users, username)
		log.Printf("User %s removed from ConnectionManager. Total users: %d", username, len(cm.users))
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

	var toRemove []string

	for username, conn := range cm.connections {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("Broadcast error: failed to send message to user %s: %v", username, err)
			toRemove = append(toRemove, username)
		}
	}

	if len(toRemove) > 0 {
		cm.mu.RUnlock()
		cm.mu.Lock()
		for _, username := range toRemove {
			if conn, exists := cm.connections[username]; exists {
				conn.Close()
				delete(cm.connections, username)
				delete(cm.users, username)
				log.Printf("Removed user %s due to broadcast error", username)
			}
		}
		cm.mu.Unlock()
		cm.mu.RLock()
	}
}

func (cm *ConnectionManager) BroadcastToUser(username string, message models.WSMessage) error {
	cm.mu.RLock()
	conn, exists := cm.connections[username]
	cm.mu.RUnlock()

	if !exists {
		return nil // User not connected, silently ignore
	}

	err := conn.WriteJSON(message)
	if err != nil {
		log.Printf("Broadcast error: failed to send message to user %s: %v", username, err)
		cm.RemoveUser(username)
		return err
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

	conn, exists := cm.connections[username]
	return conn, exists
}

func (cm *ConnectionManager) IsUserOnline(username string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, exists := cm.users[username]
	return exists
}

package client

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/ifrunruhin12/tasktime/internal/models"
	"github.com/ifrunruhin12/tasktime/internal/storage"
)

type Client struct {
	serverURL     string
	configManager *ConfigManager
	config        *ClientConfig
}

func New(serverURL string) *Client {
	cm, _ := NewConfigManager()
	return &Client{
		serverURL:     serverURL,
		configManager: cm,
	}
}

func (c *Client) Start() error {
	LogInfo("Client start requested")
	
	// Check if config exists
	if c.configManager != nil && c.configManager.Exists() {
		LogInfo("Existing config found, attempting to load")
		
		// Load existing config
		config, err := c.configManager.Load()
		if err == nil && config.AutoConnect && config.AuthToken != "" {
			LogInfo("Auto-connect enabled with stored token", "username", config.Username, "server_url", config.ServerURL)
			
			// Use config values
			c.config = config
			c.serverURL = config.ServerURL

			// Validate token and start main app
			if c.validateToken(config.AuthToken) {
				LogInfo("Token validation successful, starting main application")
				p := tea.NewProgram(c.initialModel(), tea.WithAltScreen())
				_, err := p.Run()
				return err
			} else {
				LogWarn("Token validation failed, proceeding to setup")
			}
		} else {
			LogInfo("Config loaded but auto-connect disabled or no token", "auto_connect", config != nil && config.AutoConnect, "has_token", config != nil && config.AuthToken != "")
		}
	} else {
		LogInfo("No existing config found")
	}

	// No valid config, run setup flow
	LogInfo("Starting setup flow")
	return c.runSetup()
}

func (c *Client) runSetup() error {
	LogInfo("Running setup flow")
	
	if c.configManager == nil {
		cm, err := NewConfigManager()
		if err != nil {
			LogError("Failed to create config manager", "error", err.Error())
			return err
		}
		c.configManager = cm
	}

	sm := newSetupModel(c.configManager)
	p := tea.NewProgram(sm, tea.WithAltScreen())

	_, err := p.Run()
	if err != nil {
		LogError("Setup flow failed", "error", err.Error())
		return err
	}

	// After setup completes, load the config and start main app
	if c.configManager.Exists() {
		config, err := c.configManager.Load()
		if err == nil && config.AuthToken != "" {
			LogInfo("Setup completed successfully, starting main application", "username", config.Username)
			c.config = config
			c.serverURL = config.ServerURL

			// Start main application
			p := tea.NewProgram(c.initialModel(), tea.WithAltScreen())
			_, err := p.Run()
			return err
		} else {
			LogWarn("Setup completed but no valid config found", "config_exists", c.configManager.Exists(), "has_token", config != nil && config.AuthToken != "")
		}
	}

	return nil
}

func (c *Client) validateToken(token string) bool {
	if token == "" {
		LogWarn("Token validation failed: empty token")
		return false
	}

	LogDebug("Validating token with server", "server_url", c.serverURL)

	// Try to validate token with the server
	req, err := http.NewRequest("GET", c.serverURL+"/api/v1/users/me", nil)
	if err != nil {
		LogError("Failed to create token validation request", "error", err.Error())
		return false
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		LogError("Token validation request failed", "error", err.Error())
		return false
	}
	defer resp.Body.Close()

	success := resp.StatusCode == 200
	if success {
		LogInfo("Token validation successful")
	} else {
		LogWarn("Token validation failed", "status_code", resp.StatusCode)
	}

	return success
}

func (c *Client) initialModel() model {
	localStore, _ := storage.NewLocalStore()
	return model{
		client:         c,
		personalTasks:  []models.Task{},
		teamTasks:      []models.Task{},
		cursor:         0,
		showInput:      false,
		width:          80,
		height:         24,
		currentSection: "personal", // Start with personal tasks
		localStore:     localStore,
		config:         c.config,
		currentFilter:  "all", // Default to showing all tasks
	}
}

type model struct {
	client         *Client
	personalTasks  []models.Task
	teamTasks      []models.Task
	cursor         int
	showInput      bool
	inputTitle     string
	inputProject   string
	inputMode      int
	ws             *websocket.Conn
	width          int
	height         int
	currentSection string // "personal" or "team"
	localStore     *storage.LocalStore
	config         *ClientConfig
	onlineUsers    int // Count of online users

	// Assignment UI state
	showAssignment   bool
	assignmentCursor int
	assignmentUsers  []string
	assigningTaskID  string

	// Filter state
	currentFilter string // "my" or "all"

	// Users list UI state
	showUsersList bool
	usersListData []string

	// Error handling state
	errorMessage string    // Current error message to display
	errorExpiry  time.Time // When the error message should be cleared

	// Reconnection state
	reconnectAttempts int  // Number of reconnection attempts
	isReconnecting    bool // Whether we're currently trying to reconnect
}

type personalTasksLoadedMsg []models.Task
type teamTasksLoadedMsg []models.Task
type wsConnectedMsg *websocket.Conn
type tickMsg time.Time
type taskCreationFailedMsg struct{}
type wsDisconnectedMsg struct{}
type wsConnectionFailedMsg struct{}
type wsRetryMsg struct{}
type taskOperationFailedMsg struct{}
type onlineUsersLoadedMsg []string
type taskAssignedMsg struct{}
type usersListLoadedMsg []string
type clearErrorMsg struct{}
type errorMsg struct {
	message string
}
type clientAuthErrorMsg struct {
	message string
}
type reconnectMsg struct {
	delay time.Duration
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadPersonalTasks(),
		m.loadTeamTasks(),
		m.connectWebSocket(),
		m.tick(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.showInput {
			return m.handleInputKeys(msg)
		}
		if m.showAssignment {
			return m.handleAssignmentKeys(msg)
		}
		if m.showUsersList {
			return m.handleUsersListKeys(msg)
		}
		return m.handleNormalKeys(msg)

	case personalTasksLoadedMsg:
		m.personalTasks = []models.Task(msg)
		return m, nil

	case teamTasksLoadedMsg:
		m.teamTasks = []models.Task(msg)
		return m, nil

	case wsConnectedMsg:
		LogWebSocketEvent("connected", "server_url", m.client.serverURL)
		m.ws = msg
		m.isReconnecting = false
		m.reconnectAttempts = 0
		m.errorMessage = "" // Clear any reconnection error message
		return m, m.listenWebSocket()

	case tickMsg:
		return m, m.tick()

	case models.WSMessage:
		return m.handleWebSocketMessage(msg)

	case taskCreationFailedMsg:
		// Reload tasks as fallback when creation fails
		if m.currentSection == "personal" {
			return m, m.loadPersonalTasks()
		}
		return m, m.loadTeamTasks()

	case wsDisconnectedMsg:
		LogWebSocketEvent("disconnected", "reconnect_attempts", m.reconnectAttempts)
		// WebSocket disconnected, try to reconnect with exponential backoff
		m.ws = nil
		m.isReconnecting = true
		delay := m.calculateBackoffDelay()
		m.errorMessage = fmt.Sprintf("Connection lost. Reconnecting in %v...", delay.Round(time.Second))
		m.errorExpiry = time.Now().Add(delay + time.Second)
		return m, tea.Tick(delay, func(t time.Time) tea.Msg {
			return reconnectMsg{delay: delay}
		})

	case wsConnectionFailedMsg:
		LogWebSocketEvent("connection_failed", "reconnect_attempts", m.reconnectAttempts)
		// WebSocket connection failed, try again with exponential backoff
		m.ws = nil
		m.isReconnecting = true
		delay := m.calculateBackoffDelay()
		m.errorMessage = fmt.Sprintf("Connection failed. Retrying in %v...", delay.Round(time.Second))
		m.errorExpiry = time.Now().Add(delay + time.Second)
		return m, tea.Tick(delay, func(t time.Time) tea.Msg {
			return reconnectMsg{delay: delay}
		})

	case wsRetryMsg:
		// Legacy retry message - redirect to reconnect
		return m, m.connectWebSocket()

	case reconnectMsg:
		LogWebSocketEvent("reconnect_attempt", "attempt", m.reconnectAttempts+1, "delay", msg.delay)
		// Attempt reconnection
		m.reconnectAttempts++
		return m, m.connectWebSocket()

	case taskOperationFailedMsg:
		// Task operation failed, reload tasks to get current state
		if m.currentSection == "personal" {
			return m, m.loadPersonalTasks()
		}
		return m, m.loadTeamTasks()

	case onlineUsersLoadedMsg:
		m.assignmentUsers = []string(msg)
		m.showAssignment = true
		m.assignmentCursor = 0
		return m, nil

	case taskAssignedMsg:
		// Task assigned successfully, reload tasks and close assignment screen
		m.showAssignment = false
		return m, m.loadTeamTasks()

	case usersListLoadedMsg:
		m.usersListData = []string(msg)
		m.showUsersList = true
		return m, nil

	case errorMsg:
		m.errorMessage = msg.message
		m.errorExpiry = time.Now().Add(5 * time.Second)
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return clearErrorMsg{}
		})

	case clearErrorMsg:
		// Only clear if the error has expired
		if time.Now().After(m.errorExpiry) {
			m.errorMessage = ""
		}
		return m, nil

	case clientAuthErrorMsg:
		LogAuthEvent("authentication_error", "message", msg.message)
		// Handle authentication errors - clear token and show error
		m.errorMessage = msg.message
		m.errorExpiry = time.Now().Add(5 * time.Second)
		// Clear stored token
		if m.config != nil {
			m.config.AuthToken = ""
			if m.client.configManager != nil {
				m.client.configManager.Save(m.config)
			}
		}
		// Close WebSocket connection if open
		if m.ws != nil {
			m.ws.Close()
			m.ws = nil
		}
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return clearErrorMsg{}
		})
	}

	return m, nil
}

func (m model) View() string {
	if m.showInput {
		return m.renderInputMode()
	}

	if m.showAssignment {
		return m.renderAssignmentMode()
	}

	if m.showUsersList {
		return m.renderUsersListMode()
	}

	var s strings.Builder

	// Display error banner at top if there's an error
	if m.errorMessage != "" {
		s.WriteString(errorStyle.Render("⚠ " + m.errorMessage))
		s.WriteString("\n\n")
	}

	// Enhanced status bar with username, online users count, and connection status
	statusBar := "TaskTime v1.0.0"

	// Show username if available
	if m.config != nil && m.config.Username != "" {
		statusBar += " | Connected as: " + m.config.Username
	}

	// Show online users count
	if m.onlineUsers > 0 {
		statusBar += fmt.Sprintf(" | Online: %d users", m.onlineUsers)
	}

	// Show connection status
	if m.ws != nil {
		statusBar += " | Server: LIVE"
	} else if m.isReconnecting {
		statusBar += " | Server: RECONNECTING..."
	} else {
		statusBar += " | Server: OFFLINE"
	}

	// Show current filter for team tasks
	if m.currentSection == "team" {
		if m.currentFilter == "my" {
			statusBar += " | Filter: My Tasks"
		} else {
			statusBar += " | Filter: All Tasks"
		}
	}

	s.WriteString(titleStyle.Render(statusBar))
	s.WriteString("\n\n")

	// Section tabs
	personalTab := "Personal Tasks"
	teamTab := "Team Tasks"

	if m.currentSection == "personal" {
		personalTab = "▶ " + personalTab + " ◀"
		teamTab = "  " + teamTab + "  "
		s.WriteString(selectedStyle.Render(personalTab))
		s.WriteString("   ")
		s.WriteString(normalStyle.Render(teamTab))
	} else {
		personalTab = "  " + personalTab + "  "
		teamTab = "▶ " + teamTab + " ◀"
		s.WriteString(normalStyle.Render(personalTab))
		s.WriteString("   ")
		s.WriteString(selectedStyle.Render(teamTab))
	}
	s.WriteString("\n\n")

	// Get current tasks based on section
	currentTasks := m.personalTasks
	if m.currentSection == "team" {
		currentTasks = m.teamTasks
	}

	if len(currentTasks) == 0 {
		s.WriteString("No tasks yet. Press 'n' to create one!\n\n")
	} else {
		for i, task := range currentTasks {
			line := m.renderTaskLine(i, task)
			if m.cursor == i {
				s.WriteString(selectedStyle.Render(line))
			} else {
				s.WriteString(normalStyle.Render(line))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	helpText := "tab: switch • n: new • d: done • s: timer • u: users • x: delete • r: refresh • q: quit"
	if m.currentSection == "team" {
		helpText = "tab: switch • n: new • d: done • s: timer • a: assign • f: filter • u: users • x: delete • r: refresh • q: quit"
	}
	s.WriteString(helpStyle.Render(helpText))

	return s.String()
}

// calculateBackoffDelay returns the delay for the next reconnection attempt
// using exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
func (m model) calculateBackoffDelay() time.Duration {
	baseDelay := time.Second
	maxDelay := 30 * time.Second

	// Calculate exponential delay: 2^attempts seconds
	delay := min(
		// Cap at maximum delay
		baseDelay*time.Duration(1<<uint(m.reconnectAttempts)), maxDelay)

	return delay
}

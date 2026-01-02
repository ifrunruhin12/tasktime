package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/ifrunruhin12/tasktime/internal/models"
)

// Personal task operations (local)
func (m model) loadPersonalTasks() tea.Cmd {
	return func() tea.Msg {
		if m.localStore == nil {
			return personalTasksLoadedMsg([]models.Task{})
		}

		tasks, err := m.localStore.GetTasks()
		if err != nil {
			return personalTasksLoadedMsg([]models.Task{})
		}
		return personalTasksLoadedMsg(tasks)
	}
}

func (m model) createPersonalTask(title, project string) tea.Cmd {
	return func() tea.Msg {
		if m.localStore == nil {
			return taskCreationFailedMsg{}
		}

		_, err := m.localStore.CreateTask(title, project)
		if err != nil {
			return taskCreationFailedMsg{}
		}
		return m.loadPersonalTasks()()
	}
}

func (m model) updatePersonalTaskStatus(id, status string) tea.Cmd {
	return func() tea.Msg {
		if m.localStore == nil {
			return taskOperationFailedMsg{}
		}

		_, err := m.localStore.UpdateTaskStatus(id, status)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		return m.loadPersonalTasks()()
	}
}

func (m model) deletePersonalTask(id string) tea.Cmd {
	return func() tea.Msg {
		if m.localStore == nil {
			return taskOperationFailedMsg{}
		}

		err := m.localStore.DeleteTask(id)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		return m.loadPersonalTasks()()
	}
}

func (m model) startPersonalTimer(id string) tea.Cmd {
	return func() tea.Msg {
		if m.localStore == nil {
			return taskOperationFailedMsg{}
		}

		_, err := m.localStore.StartTimer(id)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		return m.loadPersonalTasks()()
	}
}

func (m model) stopPersonalTimer(id string) tea.Cmd {
	return func() tea.Msg {
		if m.localStore == nil {
			return taskOperationFailedMsg{}
		}

		_, err := m.localStore.StopTimer(id)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		return m.loadPersonalTasks()()
	}
}

func (m model) loadTeamTasks() tea.Cmd {
	return func() tea.Msg {
		// Build URL with filter query parameter
		url := m.client.serverURL + "/api/v1/tasks"
		if m.currentFilter == "my" {
			url += "?filter=my"
		} else {
			url += "?filter=all"
		}

		LogDebug("Loading team tasks", "url", url, "filter", m.currentFilter)

		// Create request with authentication header
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			LogError("Failed to create team tasks request", "error", err.Error())
			return teamTasksLoadedMsg([]models.Task{})
		}

		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}

		startTime := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		
		if err != nil {
			LogError("Team tasks request failed", "error", err.Error(), "duration_ms", duration.Milliseconds())
			return teamTasksLoadedMsg([]models.Task{})
		}
		defer resp.Body.Close()

		LogHTTPRequest("GET", url, resp.StatusCode, duration.Milliseconds())

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			LogAuthEvent("token_expired", "endpoint", "/api/v1/tasks")
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		var tasks []models.Task
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			LogError("Failed to decode team tasks response", "error", err.Error())
			return teamTasksLoadedMsg([]models.Task{})
		}

		LogInfo("Team tasks loaded successfully", "count", len(tasks))
		return teamTasksLoadedMsg(tasks)
	}
}

func (m model) createTeamTask(title, project string) tea.Cmd {
	return func() tea.Msg {
		LogInfo("Creating team task", "title", title, "project", project)
		
		reqBody := models.CreateTaskRequest{
			Title:   title,
			Project: project,
		}

		jsonData, _ := json.Marshal(reqBody)

		// Create request with authentication header
		req, err := http.NewRequest("POST", m.client.serverURL+"/api/v1/tasks", bytes.NewBuffer(jsonData))
		if err != nil {
			LogError("Failed to create team task request", "error", err.Error())
			return taskCreationFailedMsg{}
		}

		req.Header.Set("Content-Type", "application/json")

		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}

		startTime := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		
		if err != nil {
			LogError("Team task creation request failed", "error", err.Error(), "duration_ms", duration.Milliseconds())
			return taskCreationFailedMsg{}
		}
		defer resp.Body.Close()

		LogHTTPRequest("POST", m.client.serverURL+"/api/v1/tasks", resp.StatusCode, duration.Milliseconds())

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			LogAuthEvent("token_expired", "endpoint", "/api/v1/tasks")
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			LogError("Team task creation failed", "status_code", resp.StatusCode)
			return taskCreationFailedMsg{}
		}

		LogInfo("Team task created successfully")
		return m.loadTeamTasks()()
	}
}

func (m model) updateTeamTaskStatus(id, status string) tea.Cmd {
	return func() tea.Msg {
		LogInfo("Updating team task status", "task_id", id, "status", status)
		
		reqBody := models.UpdateStatusRequest{Status: status}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("PUT",
			m.client.serverURL+"/api/v1/tasks/"+id+"/status",
			bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}

		startTime := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		
		if err != nil {
			LogError("Team task status update failed", "error", err.Error(), "task_id", id, "duration_ms", duration.Milliseconds())
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		LogHTTPRequest("PUT", m.client.serverURL+"/api/v1/tasks/"+id+"/status", resp.StatusCode, duration.Milliseconds())

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			LogAuthEvent("token_expired", "endpoint", "/api/v1/tasks/"+id+"/status")
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			LogError("Team task status update failed", "status_code", resp.StatusCode, "task_id", id)
			return taskOperationFailedMsg{}
		}

		LogInfo("Team task status updated successfully", "task_id", id, "status", status)
		return nil // WebSocket will handle the update
	}
}

func (m model) deleteTeamTask(id string) tea.Cmd {
	return func() tea.Msg {
		LogInfo("Deleting team task", "task_id", id)
		
		req, _ := http.NewRequest("DELETE", m.client.serverURL+"/api/v1/tasks/"+id, nil)

		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}

		startTime := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		
		if err != nil {
			LogError("Team task deletion failed", "error", err.Error(), "task_id", id, "duration_ms", duration.Milliseconds())
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		LogHTTPRequest("DELETE", m.client.serverURL+"/api/v1/tasks/"+id, resp.StatusCode, duration.Milliseconds())

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			LogAuthEvent("token_expired", "endpoint", "/api/v1/tasks/"+id)
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 204 {
			LogError("Team task deletion failed", "status_code", resp.StatusCode, "task_id", id)
			return taskOperationFailedMsg{}
		}

		LogInfo("Team task deleted successfully", "task_id", id)
		return nil // WebSocket will handle the update
	}
}

func (m model) startTeamTimer(id string) tea.Cmd {
	return func() tea.Msg {
		LogInfo("Starting team timer", "task_id", id)
		
		req, _ := http.NewRequest("POST", m.client.serverURL+"/api/v1/tasks/"+id+"/time/start", nil)
		req.Header.Set("Content-Type", "application/json")

		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}

		startTime := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		
		if err != nil {
			LogError("Team timer start failed", "error", err.Error(), "task_id", id, "duration_ms", duration.Milliseconds())
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		LogHTTPRequest("POST", m.client.serverURL+"/api/v1/tasks/"+id+"/time/start", resp.StatusCode, duration.Milliseconds())

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			LogAuthEvent("token_expired", "endpoint", "/api/v1/tasks/"+id+"/time/start")
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			LogError("Team timer start failed", "status_code", resp.StatusCode, "task_id", id)
			return taskOperationFailedMsg{}
		}

		LogInfo("Team timer started successfully", "task_id", id)
		return nil // WebSocket will handle the update
	}
}

func (m model) stopTeamTimer(id string) tea.Cmd {
	return func() tea.Msg {
		LogInfo("Stopping team timer", "task_id", id)
		
		req, _ := http.NewRequest("POST", m.client.serverURL+"/api/v1/tasks/"+id+"/time/stop", nil)
		req.Header.Set("Content-Type", "application/json")

		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}

		startTime := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		
		if err != nil {
			LogError("Team timer stop failed", "error", err.Error(), "task_id", id, "duration_ms", duration.Milliseconds())
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		LogHTTPRequest("POST", m.client.serverURL+"/api/v1/tasks/"+id+"/time/stop", resp.StatusCode, duration.Milliseconds())

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			LogAuthEvent("token_expired", "endpoint", "/api/v1/tasks/"+id+"/time/stop")
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			LogError("Team timer stop failed", "status_code", resp.StatusCode, "task_id", id)
			return taskOperationFailedMsg{}
		}

		LogInfo("Team timer stopped successfully", "task_id", id)
		return nil // WebSocket will handle the update
	}
}

// WebSocket operations
func (m model) connectWebSocket() tea.Cmd {
	return func() tea.Msg {
		wsURL := "ws" + m.client.serverURL[4:] + "/api/v1/ws"
		LogInfo("Attempting WebSocket connection", "url", wsURL)
		
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			LogError("WebSocket connection failed", "error", err.Error(), "url", wsURL)
			return wsConnectionFailedMsg{}
		}

		// Send authentication message immediately after connection
		if m.config != nil && m.config.AuthToken != "" {
			LogDebug("Sending WebSocket authentication message")
			authMsg := models.WSMessage{
				Type: "auth",
				Payload: map[string]interface{}{
					"token": m.config.AuthToken,
				},
			}
			if err := conn.WriteJSON(authMsg); err != nil {
				LogError("Failed to send WebSocket auth message", "error", err.Error())
				conn.Close()
				return wsConnectionFailedMsg{}
			}
		} else {
			LogWarn("No auth token available for WebSocket connection")
		}

		LogInfo("WebSocket connection established successfully")
		return wsConnectedMsg(conn)
	}
}

func (m model) listenWebSocket() tea.Cmd {
	return func() tea.Msg {
		if m.ws == nil {
			LogWarn("Attempted to listen on nil WebSocket connection")
			return wsDisconnectedMsg{}
		}

		var msg models.WSMessage
		err := m.ws.ReadJSON(&msg)
		if err != nil {
			LogError("WebSocket read error", "error", err.Error())
			return wsDisconnectedMsg{}
		}

		LogDebug("WebSocket message received", "type", msg.Type)
		return msg
	}
}

func (m model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Authentication API calls

type AuthResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// registerUser creates a new user account and returns the auth token
func registerUser(serverURL, username, password string) (*AuthResponse, error) {
	LogAuthEvent("registration_attempt", "username", username, "server_url", serverURL)
	
	reqBody := AuthRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		LogError("Failed to marshal registration request", "error", err.Error())
		return nil, err
	}

	startTime := time.Now()
	resp, err := http.Post(
		serverURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	duration := time.Since(startTime)
	
	if err != nil {
		LogError("Registration request failed", "error", err.Error(), "duration_ms", duration.Milliseconds())
		return nil, err
	}
	defer resp.Body.Close()

	LogHTTPRequest("POST", serverURL+"/api/v1/auth/register", resp.StatusCode, duration.Milliseconds())

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		var errResp map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil {
			if errObj, ok := errResp["error"].(map[string]interface{}); ok {
				if msg, ok := errObj["message"].(string); ok {
					LogAuthEvent("registration_failed", "username", username, "error", msg)
					return nil, fmt.Errorf("%s", msg)
				}
			}
		}
		LogAuthEvent("registration_failed", "username", username, "status_code", resp.StatusCode)
		return nil, fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		LogError("Failed to decode registration response", "error", err.Error())
		return nil, err
	}

	LogAuthEvent("registration_success", "username", username)
	return &authResp, nil
}

// loginUser authenticates a user and returns the auth token
func loginUser(serverURL, username, password string) (*AuthResponse, error) {
	LogAuthEvent("login_attempt", "username", username, "server_url", serverURL)
	
	reqBody := AuthRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		LogError("Failed to marshal login request", "error", err.Error())
		return nil, err
	}

	startTime := time.Now()
	resp, err := http.Post(
		serverURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	duration := time.Since(startTime)
	
	if err != nil {
		LogError("Login request failed", "error", err.Error(), "duration_ms", duration.Milliseconds())
		return nil, err
	}
	defer resp.Body.Close()

	LogHTTPRequest("POST", serverURL+"/api/v1/auth/login", resp.StatusCode, duration.Milliseconds())

	if resp.StatusCode != 200 {
		var errResp map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil {
			if errObj, ok := errResp["error"].(map[string]interface{}); ok {
				if msg, ok := errObj["message"].(string); ok {
					LogAuthEvent("login_failed", "username", username, "error", msg)
					return nil, fmt.Errorf("%s", msg)
				}
			}
		}
		LogAuthEvent("login_failed", "username", username, "status_code", resp.StatusCode)
		return nil, fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		LogError("Failed to decode login response", "error", err.Error())
		return nil, err
	}

	LogAuthEvent("login_success", "username", username)
	return &authResp, nil
}

// loadOnlineUsers fetches the list of online users from the server
func (m model) loadOnlineUsers() tea.Cmd {
	return func() tea.Msg {
		if m.config == nil || m.config.AuthToken == "" {
			LogWarn("Cannot load online users: no auth token")
			return onlineUsersLoadedMsg([]string{})
		}

		LogDebug("Loading online users")

		req, err := http.NewRequest("GET", m.client.serverURL+"/api/v1/users/online", nil)
		if err != nil {
			LogError("Failed to create online users request", "error", err.Error())
			return onlineUsersLoadedMsg([]string{})
		}

		req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)

		startTime := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		
		if err != nil {
			LogError("Online users request failed", "error", err.Error(), "duration_ms", duration.Milliseconds())
			return onlineUsersLoadedMsg([]string{})
		}
		defer resp.Body.Close()

		LogHTTPRequest("GET", m.client.serverURL+"/api/v1/users/online", resp.StatusCode, duration.Milliseconds())

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			LogAuthEvent("token_expired", "endpoint", "/api/v1/users/online")
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			LogError("Online users request failed", "status_code", resp.StatusCode)
			return onlineUsersLoadedMsg([]string{})
		}

		var result struct {
			Users []string `json:"users"`
			Count int      `json:"count"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			LogError("Failed to decode online users response", "error", err.Error())
			return onlineUsersLoadedMsg([]string{})
		}

		LogInfo("Online users loaded successfully", "count", result.Count)
		return onlineUsersLoadedMsg(result.Users)
	}
}

// assignTask assigns a task to a user or unassigns it
func (m model) assignTask(taskID string, assignedTo *string) tea.Cmd {
	return func() tea.Msg {
		if m.config == nil || m.config.AuthToken == "" {
			LogWarn("Cannot assign task: no auth token", "task_id", taskID)
			return taskOperationFailedMsg{}
		}

		assignedUser := "unassigned"
		if assignedTo != nil {
			assignedUser = *assignedTo
		}
		LogInfo("Assigning task", "task_id", taskID, "assigned_to", assignedUser)

		reqBody := map[string]interface{}{}
		if assignedTo != nil {
			reqBody["assigned_to"] = *assignedTo
		} else {
			reqBody["assigned_to"] = nil
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			LogError("Failed to marshal task assignment request", "error", err.Error())
			return taskOperationFailedMsg{}
		}

		req, err := http.NewRequest("PUT",
			m.client.serverURL+"/api/v1/tasks/"+taskID+"/assign",
			bytes.NewBuffer(jsonData))
		if err != nil {
			LogError("Failed to create task assignment request", "error", err.Error())
			return taskOperationFailedMsg{}
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)

		startTime := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		
		if err != nil {
			LogError("Task assignment request failed", "error", err.Error(), "task_id", taskID, "duration_ms", duration.Milliseconds())
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		LogHTTPRequest("PUT", m.client.serverURL+"/api/v1/tasks/"+taskID+"/assign", resp.StatusCode, duration.Milliseconds())

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			LogAuthEvent("token_expired", "endpoint", "/api/v1/tasks/"+taskID+"/assign")
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			LogError("Task assignment failed", "status_code", resp.StatusCode, "task_id", taskID)
			return taskOperationFailedMsg{}
		}

		LogInfo("Task assigned successfully", "task_id", taskID, "assigned_to", assignedUser)
		return taskAssignedMsg{}
	}
}

// loadUsersListData fetches the list of online users for the users list screen
func (m model) loadUsersListData() tea.Cmd {
	return func() tea.Msg {
		if m.config == nil || m.config.AuthToken == "" {
			return usersListLoadedMsg([]string{})
		}

		req, err := http.NewRequest("GET", m.client.serverURL+"/api/v1/users/online", nil)
		if err != nil {
			return usersListLoadedMsg([]string{})
		}

		req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return usersListLoadedMsg([]string{})
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			return usersListLoadedMsg([]string{})
		}

		var result struct {
			Users []string `json:"users"`
			Count int      `json:"count"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return usersListLoadedMsg([]string{})
		}

		return usersListLoadedMsg(result.Users)
	}
}

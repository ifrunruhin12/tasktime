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

		// Create request with authentication header
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return teamTasksLoadedMsg([]models.Task{})
		}

		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return teamTasksLoadedMsg([]models.Task{})
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		var tasks []models.Task
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			return teamTasksLoadedMsg([]models.Task{})
		}

		return teamTasksLoadedMsg(tasks)
	}
}

func (m model) createTeamTask(title, project string) tea.Cmd {
	return func() tea.Msg {
		reqBody := models.CreateTaskRequest{
			Title:   title,
			Project: project,
		}

		jsonData, _ := json.Marshal(reqBody)
		
		// Create request with authentication header
		req, err := http.NewRequest("POST", m.client.serverURL+"/api/v1/tasks", bytes.NewBuffer(jsonData))
		if err != nil {
			return taskCreationFailedMsg{}
		}
		
		req.Header.Set("Content-Type", "application/json")
		
		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return taskCreationFailedMsg{}
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			return taskCreationFailedMsg{}
		}

		return m.loadTeamTasks()()
	}
}

func (m model) updateTeamTaskStatus(id, status string) tea.Cmd {
	return func() tea.Msg {
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

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}

		return nil // WebSocket will handle the update
	}
}

func (m model) deleteTeamTask(id string) tea.Cmd {
	return func() tea.Msg {
		req, _ := http.NewRequest("DELETE", m.client.serverURL+"/api/v1/tasks/"+id, nil)
		
		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 204 {
			return taskOperationFailedMsg{}
		}

		return nil // WebSocket will handle the update
	}
}

func (m model) startTeamTimer(id string) tea.Cmd {
	return func() tea.Msg {
		req, _ := http.NewRequest("POST", m.client.serverURL+"/api/v1/tasks/"+id+"/time/start", nil)
		req.Header.Set("Content-Type", "application/json")
		
		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}

		return nil // WebSocket will handle the update
	}
}

func (m model) stopTeamTimer(id string) tea.Cmd {
	return func() tea.Msg {
		req, _ := http.NewRequest("POST", m.client.serverURL+"/api/v1/tasks/"+id+"/time/stop", nil)
		req.Header.Set("Content-Type", "application/json")
		
		// Add authentication token if available
		if m.config != nil && m.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)
		}
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}

		return nil // WebSocket will handle the update
	}
}

// WebSocket operations
func (m model) connectWebSocket() tea.Cmd {
	return func() tea.Msg {
		wsURL := "ws" + m.client.serverURL[4:] + "/api/v1/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			return wsConnectionFailedMsg{}
		}

		// Send authentication message immediately after connection
		if m.config != nil && m.config.AuthToken != "" {
			authMsg := models.WSMessage{
				Type: "auth",
				Payload: map[string]interface{}{
					"token": m.config.AuthToken,
				},
			}
			if err := conn.WriteJSON(authMsg); err != nil {
				conn.Close()
				return wsConnectionFailedMsg{}
			}
		}

		return wsConnectedMsg(conn)
	}
}

func (m model) listenWebSocket() tea.Cmd {
	return func() tea.Msg {
		if m.ws == nil {
			return wsDisconnectedMsg{}
		}

		var msg models.WSMessage
		err := m.ws.ReadJSON(&msg)
		if err != nil {
			return wsDisconnectedMsg{}
		}

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
	reqBody := AuthRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		serverURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		var errResp map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil {
			if errObj, ok := errResp["error"].(map[string]interface{}); ok {
				if msg, ok := errObj["message"].(string); ok {
					return nil, fmt.Errorf("%s", msg)
				}
			}
		}
		return nil, fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}

// loginUser authenticates a user and returns the auth token
func loginUser(serverURL, username, password string) (*AuthResponse, error) {
	reqBody := AuthRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		serverURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errResp map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil {
			if errObj, ok := errResp["error"].(map[string]interface{}); ok {
				if msg, ok := errObj["message"].(string); ok {
					return nil, fmt.Errorf("%s", msg)
				}
			}
		}
		return nil, fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}

// loadOnlineUsers fetches the list of online users from the server
func (m model) loadOnlineUsers() tea.Cmd {
	return func() tea.Msg {
		if m.config == nil || m.config.AuthToken == "" {
			return onlineUsersLoadedMsg([]string{})
		}

		req, err := http.NewRequest("GET", m.client.serverURL+"/api/v1/users/online", nil)
		if err != nil {
			return onlineUsersLoadedMsg([]string{})
		}

		req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return onlineUsersLoadedMsg([]string{})
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			return onlineUsersLoadedMsg([]string{})
		}

		var result struct {
			Users []string `json:"users"`
			Count int      `json:"count"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return onlineUsersLoadedMsg([]string{})
		}

		return onlineUsersLoadedMsg(result.Users)
	}
}

// assignTask assigns a task to a user or unassigns it
func (m model) assignTask(taskID string, assignedTo *string) tea.Cmd {
	return func() tea.Msg {
		if m.config == nil || m.config.AuthToken == "" {
			return taskOperationFailedMsg{}
		}

		reqBody := map[string]interface{}{}
		if assignedTo != nil {
			reqBody["assigned_to"] = *assignedTo
		} else {
			reqBody["assigned_to"] = nil
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return taskOperationFailedMsg{}
		}

		req, err := http.NewRequest("PUT",
			m.client.serverURL+"/api/v1/tasks/"+taskID+"/assign",
			bytes.NewBuffer(jsonData))
		if err != nil {
			return taskOperationFailedMsg{}
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.config.AuthToken)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		// Handle 401 Unauthorized - authentication error
		if resp.StatusCode == 401 {
			return clientAuthErrorMsg{message: "Session expired. Please restart the client to re-authenticate."}
		}

		if resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}

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

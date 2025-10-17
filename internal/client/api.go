package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ifrunruhin12/tasktime-mvp/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

func (m model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(m.client.serverURL + "/api/v1/tasks")
		if err != nil {
			return tasksLoadedMsg{}
		}
		defer resp.Body.Close()

		var tasks []models.Task
		json.NewDecoder(resp.Body).Decode(&tasks)
		return tasksLoadedMsg(tasks)
	}
}

func (m model) createTask(title, project string) tea.Cmd {
	return func() tea.Msg {
		payload := models.CreateTaskRequest{
			Title:   title,
			Project: project,
		}
		jsonData, _ := json.Marshal(payload)

		resp, err := http.Post(m.client.serverURL+"/api/v1/tasks", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			// If HTTP request fails, reload tasks as fallback
			return taskCreationFailedMsg{}
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			// If server returns error, reload tasks as fallback
			return taskCreationFailedMsg{}
		}

		// Don't return the task directly - let WebSocket handle it for all clients
		// This ensures consistent behavior across all connected clients
		return nil
	}
}

func (m model) updateTaskStatus(taskID, status string) tea.Cmd {
	return func() tea.Msg {
		payload := models.UpdateStatusRequest{Status: status}
		jsonData, _ := json.Marshal(payload)

		req, _ := http.NewRequest("PUT", m.client.serverURL+"/api/v1/tasks/"+taskID+"/status", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}
		
		return nil
	}
}

func (m model) startTimer(taskID string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Post(m.client.serverURL+"/api/v1/tasks/"+taskID+"/time/start", "application/json", nil)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}
		
		return nil
	}
}

func (m model) stopTimer(taskID string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Post(m.client.serverURL+"/api/v1/tasks/"+taskID+"/time/stop", "application/json", nil)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}
		
		return nil
	}
}

func (m model) deleteTask(taskID string) tea.Cmd {
	return func() tea.Msg {
		req, _ := http.NewRequest("DELETE", m.client.serverURL+"/api/v1/tasks/"+taskID, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != 204 {
			return taskOperationFailedMsg{}
		}
		
		return nil
	}
}

func (m model) connectWebSocket() tea.Cmd {
	return func() tea.Msg {
		wsURL := "ws://localhost:8080/api/v1/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			// Return nil but don't crash - WebSocket is optional for basic functionality
			return wsConnectionFailedMsg{}
		}
		return wsConnectedMsg(conn)
	}
}

func (m model) listenWebSocket() tea.Cmd {
	return func() tea.Msg {
		if m.ws == nil {
			return nil
		}
		
		var msg models.WSMessage
		err := m.ws.ReadJSON(&msg)
		if err != nil {
			// WebSocket connection lost, try to reconnect
			m.ws = nil
			return wsDisconnectedMsg{}
		}
		
		// Debug: Log received messages (you can remove this later)
		// This will help us see if messages are being received
		return msg
	}
}

func (m model) tick() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
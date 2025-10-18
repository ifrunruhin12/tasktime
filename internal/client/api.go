package client

import (
	"bytes"
	"encoding/json"
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

// Team task operations (server API)
func (m model) loadTeamTasks() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(m.client.serverURL + "/api/v1/tasks")
		if err != nil {
			return teamTasksLoadedMsg([]models.Task{})
		}
		defer resp.Body.Close()

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
		resp, err := http.Post(
			m.client.serverURL+"/api/v1/tasks",
			"application/json",
			bytes.NewBuffer(jsonData),
		)

		if err != nil || resp.StatusCode != 200 {
			return taskCreationFailedMsg{}
		}
		defer resp.Body.Close()

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

		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil || resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		return nil // WebSocket will handle the update
	}
}

func (m model) deleteTeamTask(id string) tea.Cmd {
	return func() tea.Msg {
		req, _ := http.NewRequest("DELETE", m.client.serverURL+"/api/v1/tasks/"+id, nil)
		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil || resp.StatusCode != 204 {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		return nil // WebSocket will handle the update
	}
}

func (m model) startTeamTimer(id string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Post(m.client.serverURL+"/api/v1/tasks/"+id+"/time/start", "application/json", nil)
		if err != nil || resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

		return nil // WebSocket will handle the update
	}
}

func (m model) stopTeamTimer(id string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Post(m.client.serverURL+"/api/v1/tasks/"+id+"/time/stop", "application/json", nil)
		if err != nil || resp.StatusCode != 200 {
			return taskOperationFailedMsg{}
		}
		defer resp.Body.Close()

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
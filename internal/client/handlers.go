package client

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifrunruhin12/tasktime/internal/models"
)

func (m model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	currentTasks := m.personalTasks
	if m.currentSection == "team" {
		currentTasks = m.teamTasks
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.ws != nil {
			m.ws.Close()
		}
		return m, tea.Quit

	case "tab":
		// Switch between personal and team sections
		if m.currentSection == "personal" {
			m.currentSection = "team"
		} else {
			m.currentSection = "personal"
		}
		m.cursor = 0 // Reset cursor when switching sections

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(currentTasks)-1 {
			m.cursor++
		}

	case "n":
		m.showInput = true
		m.inputTitle = ""
		m.inputProject = ""
		m.inputMode = 0

	case "d":
		if len(currentTasks) > 0 && m.cursor < len(currentTasks) {
			task := currentTasks[m.cursor]
			newStatus := "done"
			if task.Status == "done" {
				newStatus = "todo"
			}

			if m.currentSection == "personal" {
				return m, m.updatePersonalTaskStatus(task.ID, newStatus)
			} else {
				return m, m.updateTeamTaskStatus(task.ID, newStatus)
			}
		}

	case "s":
		if len(currentTasks) > 0 && m.cursor < len(currentTasks) {
			task := currentTasks[m.cursor]

			if m.currentSection == "personal" {
				if task.IsActive {
					return m, m.stopPersonalTimer(task.ID)
				} else {
					return m, m.startPersonalTimer(task.ID)
				}
			} else {
				if task.IsActive {
					return m, m.stopTeamTimer(task.ID)
				} else {
					return m, m.startTeamTimer(task.ID)
				}
			}
		}

	case "x":
		if len(currentTasks) > 0 && m.cursor < len(currentTasks) {
			task := currentTasks[m.cursor]

			if m.currentSection == "personal" {
				return m, m.deletePersonalTask(task.ID)
			} else {
				return m, m.deleteTeamTask(task.ID)
			}
		}

	case "r":
		if m.currentSection == "personal" {
			return m, m.loadPersonalTasks()
		} else {
			return m, m.loadTeamTasks()
		}

	case "a":
		// Assignment only works for team tasks
		if m.currentSection == "team" && len(currentTasks) > 0 && m.cursor < len(currentTasks) {
			task := currentTasks[m.cursor]
			m.assigningTaskID = task.ID
			// Load online users and show assignment screen
			return m, m.loadOnlineUsers()
		}

	case "f":
		// Toggle filter between "my" and "all" (only for team tasks)
		if m.currentSection == "team" {
			if m.currentFilter == "all" {
				m.currentFilter = "my"
			} else {
				m.currentFilter = "all"
			}
			// Reload tasks with new filter
			return m, m.loadTeamTasks()
		}

	case "u":
		// Show online users list
		return m, m.loadUsersListData()
	}

	return m, nil
}

func (m model) handleInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.showInput = false
		return m, nil

	case "enter":
		if m.inputMode == 0 && m.inputTitle != "" {
			m.inputMode = 1
			return m, nil
		} else if m.inputMode == 1 {
			m.showInput = false

			if m.currentSection == "personal" {
				return m, m.createPersonalTask(m.inputTitle, m.inputProject)
			} else {
				return m, m.createTeamTask(m.inputTitle, m.inputProject)
			}
		}

	case "backspace":
		if m.inputMode == 0 && len(m.inputTitle) > 0 {
			m.inputTitle = m.inputTitle[:len(m.inputTitle)-1]
		} else if m.inputMode == 1 && len(m.inputProject) > 0 {
			m.inputProject = m.inputProject[:len(m.inputProject)-1]
		}

	default:
		if m.inputMode == 0 {
			m.inputTitle += msg.String()
		} else {
			m.inputProject += msg.String()
		}
	}

	return m, nil
}

func (m model) handleUsersListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key press returns to normal view
	m.showUsersList = false
	return m, nil
}

func (m model) handleWebSocketMessage(msg models.WSMessage) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case "auth.success":
		// Authentication successful, extract online users count if available
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if onlineUsers, ok := payload["online_users"].([]interface{}); ok {
				m.onlineUsers = len(onlineUsers)
			}
		}
		return m, m.listenWebSocket()

	case "auth.failed":
		// Authentication failed, close connection and show error
		if m.ws != nil {
			m.ws.Close()
			m.ws = nil
		}
		// Return auth error message to clear token and show error
		return m, func() tea.Msg {
			return clientAuthErrorMsg{message: "WebSocket authentication failed. Please restart the client to re-authenticate."}
		}

	case "user.joined":
		// User joined, increment online users count (but not for our own join)
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if username, ok := payload["username"].(string); ok {
				// Don't increment for our own join event
				if m.config != nil && username != m.config.Username {
					m.onlineUsers++
				}
			}
		}
		return m, m.listenWebSocket()

	case "user.left":
		// User left, decrement online users count
		if m.onlineUsers > 0 {
			m.onlineUsers--
		}
		return m, m.listenWebSocket()

	case "task.created":
		taskBytes, _ := json.Marshal(msg.Payload)
		var task models.Task
		if json.Unmarshal(taskBytes, &task) == nil {
			exists := false
			for _, existingTask := range m.teamTasks {
				if existingTask.ID == task.ID {
					exists = true
					break
				}
			}
			if !exists {
				m.teamTasks = append([]models.Task{task}, m.teamTasks...)
			}
		}

	case "task.updated":
		taskBytes, _ := json.Marshal(msg.Payload)
		var updatedTask models.Task
		if json.Unmarshal(taskBytes, &updatedTask) == nil {
			for i, task := range m.teamTasks {
				if task.ID == updatedTask.ID {
					m.teamTasks[i] = updatedTask
					break
				}
			}
		}

	case "task.assigned":
		// Task assignment changed, reload task list to get updated data
		return m, m.loadTeamTasks()

	case "task.deleted":
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if taskID, ok := payload["id"].(string); ok {
				for i, task := range m.teamTasks {
					if task.ID == taskID {
						m.teamTasks = append(m.teamTasks[:i], m.teamTasks[i+1:]...)
						// Adjust cursor if we're in team section and cursor is out of bounds
						if m.currentSection == "team" && m.cursor >= len(m.teamTasks) && len(m.teamTasks) > 0 {
							m.cursor = len(m.teamTasks) - 1
						}
						break
					}
				}
			}
		}
	}

	return m, m.listenWebSocket()
}

func (m model) handleAssignmentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		// Cancel assignment
		m.showAssignment = false
		m.assigningTaskID = ""
		return m, nil

	case "up", "k":
		if m.assignmentCursor > 0 {
			m.assignmentCursor--
		}

	case "down", "j":
		// +1 because we have "Unassign" option at index 0
		maxCursor := len(m.assignmentUsers)
		if m.assignmentCursor < maxCursor {
			m.assignmentCursor++
		}

	case "enter":
		// Assign task
		if m.assigningTaskID == "" {
			m.showAssignment = false
			return m, nil
		}

		var assignedTo *string
		if m.assignmentCursor == 0 {
			// Unassign option selected
			assignedTo = nil
		} else if m.assignmentCursor > 0 && m.assignmentCursor <= len(m.assignmentUsers) {
			// User selected
			username := m.assignmentUsers[m.assignmentCursor-1]
			assignedTo = &username
		}

		return m, m.assignTask(m.assigningTaskID, assignedTo)
	}

	return m, nil
}

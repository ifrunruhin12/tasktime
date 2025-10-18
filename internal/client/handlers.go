package client

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifrunruhin12/tasktime/internal/models"
)

func (m model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Get current tasks based on section
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

func (m model) handleWebSocketMessage(msg models.WSMessage) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case "task.created":
		taskBytes, _ := json.Marshal(msg.Payload)
		var task models.Task
		if json.Unmarshal(taskBytes, &task) == nil {
			// Check if task already exists to avoid duplicates
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

package client

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ifrunruhin12/tasktime/internal/models"
)

func (m model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.ws != nil {
			m.ws.Close()
		}
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}

	case "n":
		m.showInput = true
		m.inputTitle = ""
		m.inputProject = ""
		m.inputMode = 0

	case "d":
		if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
			task := m.tasks[m.cursor]
			newStatus := "done"
			if task.Status == "done" {
				newStatus = "todo"
			}
			return m, m.updateTaskStatus(task.ID, newStatus)
		}

	case "s":
		if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
			task := m.tasks[m.cursor]
			if task.IsActive {
				return m, m.stopTimer(task.ID)
			} else {
				return m, m.startTimer(task.ID)
			}
		}

	case "x":
		if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
			task := m.tasks[m.cursor]
			return m, m.deleteTask(task.ID)
		}

	case "r":
		return m, m.loadTasks()
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
			return m, m.createTask(m.inputTitle, m.inputProject)
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
			for _, existingTask := range m.tasks {
				if existingTask.ID == task.ID {
					exists = true
					break
				}
			}
			if !exists {
				m.tasks = append([]models.Task{task}, m.tasks...)
			}
		}

	case "task.updated":
		taskBytes, _ := json.Marshal(msg.Payload)
		var updatedTask models.Task
		if json.Unmarshal(taskBytes, &updatedTask) == nil {
			for i, task := range m.tasks {
				if task.ID == updatedTask.ID {
					m.tasks[i] = updatedTask
					break
				}
			}
		}

	case "task.deleted":
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if taskID, ok := payload["id"].(string); ok {
				for i, task := range m.tasks {
					if task.ID == taskID {
						m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
						if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
							m.cursor = len(m.tasks) - 1
						}
						break
					}
				}
			}
		}
	}

	return m, m.listenWebSocket()
}

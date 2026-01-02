package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/ifrunruhin12/tasktime/internal/models"
)

func (m model) renderInputMode() string {
	var s strings.Builder

	sectionName := "Personal"
	if m.currentSection == "team" {
		sectionName = "Team"
	}

	s.WriteString(titleStyle.Render(fmt.Sprintf("Create New %s Task", sectionName)))
	s.WriteString("\n\n")

	if m.inputMode == 0 {
		s.WriteString(fmt.Sprintf("Title: %s█\n", m.inputTitle))
		s.WriteString("Project: \n\n")
	} else {
		s.WriteString(fmt.Sprintf("Title: %s\n", m.inputTitle))
		s.WriteString(fmt.Sprintf("Project: %s█\n\n", m.inputProject))
	}

	s.WriteString(helpStyle.Render("Enter to continue • Esc to cancel"))
	return s.String()
}

func (m model) renderTaskLine(index int, task models.Task) string {
	cursor := "  "
	if m.cursor == index {
		cursor = "▶ "
	}

	status := "○"
	if task.Status == "done" {
		status = "●"
	}

	totalSeconds := task.TotalTimeSeconds
	if task.IsActive && task.StartTime != nil && !task.StartTime.IsZero() {
		currentSession := time.Since(*task.StartTime)
		totalSeconds += int(currentSession.Seconds())
	}

	timer := ""
	if totalSeconds > 0 || task.IsActive {
		hours := totalSeconds / 3600
		minutes := (totalSeconds % 3600) / 60
		seconds := totalSeconds % 60

		if hours > 0 {
			timer = fmt.Sprintf(" %02d:%02d:%02d", hours, minutes, seconds)
		} else {
			timer = fmt.Sprintf(" %02d:%02d", minutes, seconds)
		}

		if task.IsActive {
			timer += " ▶"
		}
	}

	project := ""
	if task.Project != "" {
		project = fmt.Sprintf(" [%s]", task.Project)
	}

	// Show assigned user for team tasks
	assignedUser := ""
	if task.AssignedTo != nil && *task.AssignedTo != "" {
		assignedUser = fmt.Sprintf(" [@%s]", *task.AssignedTo)
	}

	return fmt.Sprintf("%s%s %s%s%s%s", cursor, status, task.Title, project, assignedUser, timer)
}

func (m model) renderAssignmentMode() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Assign Task"))
	s.WriteString("\n\n")

	// Show the task being assigned
	if m.assigningTaskID != "" {
		var taskTitle string
		for _, task := range m.teamTasks {
			if task.ID == m.assigningTaskID {
				taskTitle = task.Title
				if task.Project != "" {
					taskTitle += fmt.Sprintf(" [%s]", task.Project)
				}
				break
			}
		}
		if taskTitle != "" {
			s.WriteString(fmt.Sprintf("Task: %s\n\n", taskTitle))
		}
	}

	s.WriteString("Assign to:\n")

	// Add "Unassign" option at the top
	unassignOption := "[Unassign]"
	if m.assignmentCursor == 0 {
		s.WriteString(selectedStyle.Render("▶ " + unassignOption))
	} else {
		s.WriteString(normalStyle.Render("  " + unassignOption))
	}
	s.WriteString("\n")

	// List online users
	for i, username := range m.assignmentUsers {
		userLine := username
		if m.config != nil && username == m.config.Username {
			userLine += " (you)"
		}

		if m.assignmentCursor == i+1 {
			s.WriteString(selectedStyle.Render("▶ " + userLine))
		} else {
			s.WriteString(normalStyle.Render("  " + userLine))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("↑/↓: navigate • Enter: select • Esc: cancel"))

	return s.String()
}

func (m model) renderUsersListMode() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Online Users"))
	s.WriteString("\n\n")

	if len(m.usersListData) == 0 {
		s.WriteString("No users online\n\n")
	} else {
		// Display all online users
		for _, username := range m.usersListData {
			userLine := "● " + username
			// Highlight current user
			if m.config != nil && username == m.config.Username {
				userLine += " (you)"
			}
			s.WriteString(normalStyle.Render(userLine))
			s.WriteString("\n")
		}
		s.WriteString("\n")

		// Show total count
		s.WriteString(fmt.Sprintf("Total: %d users online\n\n", len(m.usersListData)))
	}

	s.WriteString(helpStyle.Render("Press any key to return"))

	return s.String()
}

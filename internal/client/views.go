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

	// Calculate total time including current session
	totalSeconds := task.TotalTimeSeconds
	if task.IsActive && task.StartTime != nil && !task.StartTime.IsZero() {
		currentSession := time.Since(*task.StartTime)
		totalSeconds += int(currentSession.Seconds())
	}

	// Format time display
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

	return fmt.Sprintf("%s%s %s%s%s", cursor, status, task.Title, project, timer)
}
